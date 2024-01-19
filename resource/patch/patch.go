package patch

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/I-Am-Dench/lu-launcher/client"
	"github.com/I-Am-Dench/lu-launcher/ldf"
	"github.com/I-Am-Dench/lu-launcher/resource/server"
)

type Patch interface {
	DownloadResources(*server.Server, RejectionList) error
	RunWithDependencies(*server.Server, RejectionList) error

	TransferResources(clientDirectory string, cache client.Cache, server *server.Server) error
}

type tppPatch struct {
	Version string `json:"-"`
	dir     string `json:"-"`

	Dependencies []string `json:"depend,omitempty"`

	Download []struct {
		Path string `json:"path"`
		Name string `json:"name"`
	} `json:"download,omitempty"`

	Update struct {
		Boot     string `json:"boot,omitempty"`
		Protocol string `json:"protocol,omitempty"`
	} `json:"update,omitempty"`

	Replace map[string]string `json:"replace,omitempty"`
	Add     map[string]string `json:"add,omitempty"`
}

func (p *tppPatch) getPatch(version string, server *server.Server) (*tppPatch, error) {
	patchDirectory := filepath.Join(p.dir, server.Id, version)
	path := filepath.Join(patchDirectory, "patch.json")

	data, err := os.ReadFile(path)
	if err == nil {
		patch := &tppPatch{Version: version}

		err = json.Unmarshal(data, &patch)
		if err != nil {
			return nil, fmt.Errorf("cannot unmarshal \"%s\": %v", path, err)
		}

		return patch, nil
	}

	response, err := server.RemoteGet(version)
	if err != nil {
		return nil, ErrPatchesUnavailable
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusUnauthorized {
		return nil, ErrPatchesUnauthorized
	}

	if response.StatusCode >= 400 {
		return nil, ErrPatchesUnavailable
	}

	data, err = io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("cannot read body of patch version response: %v", err)
	}

	patch := &tppPatch{Version: version}
	err = json.Unmarshal(data, &patch)
	if err != nil {
		return nil, fmt.Errorf("malformed response body from patch version: %v", err)
	}

	os.MkdirAll(patchDirectory, 0755)

	err = os.WriteFile(path, data, 0755)
	if err != nil {
		log.Printf("Could not save patch.json: %v", err)
	}

	return patch, err
}

func (patch *tppPatch) doDownload(server *server.Server) error {
	log.Println("Starting downloads...")
	path := filepath.Join(patch.dir, server.Id, patch.Version)
	os.MkdirAll(path, 0755)

	for _, download := range patch.Download {
		if !filepath.IsLocal(download.Name) {
			return &PatchError{fmt.Errorf("invalid download name \"%s\": name is nonlocal", download.Name)}
		}

		response, err := server.RemoteGet(download.Name)
		if err != nil {
			return &PatchError{fmt.Errorf("could not get url: %v", err)}
		}
		defer response.Body.Close()

		if response.StatusCode == http.StatusUnauthorized {
			return &PatchError{ErrPatchesUnauthorized}
		}

		if response.StatusCode >= 400 {
			return &PatchError{fmt.Errorf("invalid response status code from server: %d", response.StatusCode)}
		}

		data, err := io.ReadAll(response.Body)
		if err != nil {
			return &PatchError{fmt.Errorf("could not read body of server response: %v", err)}
		}

		err = os.WriteFile(filepath.Join(path, download.Name), data, 0755)
		if err != nil {
			return &PatchError{fmt.Errorf("could not save download \"%s\" to \"%s\": %v", download.Path, download.Name, err)}
		}
	}

	return nil
}

func (patch *tppPatch) DownloadResources(server *server.Server, rejections RejectionList) error {
	if rejections.IsRejected(server, patch.Version) {
		return &PatchError{fmt.Errorf("\"%s\" is rejected", patch.Version)}
	}

	if err := ValidateVersionName(patch.Version); err != nil {
		return &PatchError{fmt.Errorf("%v", err)}
	}

	return patch.doDownload(server)
}

func (patch *tppPatch) parseDependencyVersion(version string) (string, bool) {
	trimmed := strings.TrimSpace(version)
	if len(trimmed) > 0 && trimmed[len(trimmed)-1] == '*' {
		return trimmed[:len(trimmed)-1], true
	}
	return trimmed, false
}

func (patch *tppPatch) getDependencies(server *server.Server, recursive ...bool) ([]*tppPatch, error) {
	recurse := false
	if len(recursive) > 0 {
		recurse = recursive[0]
	}

	patches := []*tppPatch{}

	for _, dependencyName := range patch.Dependencies {
		version, fetchSubDependencies := patch.parseDependencyVersion(dependencyName)
		if len(version) == 0 {
			continue
		}

		dependency, err := patch.getPatch(version, server)
		if err != nil {
			return []*tppPatch{}, fmt.Errorf("cannot resolve patch dependency \"%s\": %v", version, err)
		}

		patches = append(patches, dependency)
		if fetchSubDependencies || recurse {
			subDependencies, err := dependency.getDependencies(server, recurse)
			if err != nil {
				return []*tppPatch{}, fmt.Errorf("cannot resolve recursive dependency \"%s\": %v", version, err)
			}
			patches = append(patches, subDependencies...)
		}
	}

	return patches, nil
}

func (patch *tppPatch) updateBoot(server *server.Server) error {
	if len(patch.Update.Boot) == 0 {
		return nil
	}

	log.Println("Updating boot file...")
	path := filepath.Join(patch.dir, server.Id, patch.Version)

	data, err := os.ReadFile(filepath.Join(path, patch.Update.Boot))
	if err != nil {
		return fmt.Errorf("could not read boot patch file \"%s\": %v", patch.Update.Boot, err)
	}

	config := &ldf.BootConfig{}
	err = ldf.Unmarshal(data, config)
	if err != nil {
		return fmt.Errorf("could not unmarshal boot patch file: %v", err)
	}

	server.Config = config
	return server.SaveConfig()
}

func (patch *tppPatch) updateProtocol(server *server.Server) error {
	if len(patch.Update.Protocol) == 0 {
		return nil
	}

	log.Printf("Updating patch server protocol to \"%s\"", patch.Update.Protocol)
	server.PatchProtocol = patch.Update.Protocol

	return nil
}

func (patch *tppPatch) doUpdates(server *server.Server) error {
	return errors.Join(
		patch.updateBoot(server),
		patch.updateProtocol(server),
	)
}

func (patch *tppPatch) RunWithDependencies(server *server.Server, rejections RejectionList) error {
	dependencies, err := patch.getDependencies(server)
	if err != nil {
		return err
	}

	for _, dependency := range dependencies {
		err := dependency.DownloadResources(server, rejections)
		if err != nil {
			return &PatchError{fmt.Errorf("run dependency \"%s\": %v", dependency.Version, err)}
		}
	}

	err = patch.DownloadResources(server, rejections)
	if err != nil {
		return &PatchError{err}
	}

	return patch.doUpdates(server)
}

func (patch *tppPatch) replaceResources(clientDirectory string, cache client.Cache, server *server.Server) error {
	for source, destination := range patch.Replace {
		if !filepath.IsLocal(source) {
			return fmt.Errorf("invalid source resource \"%s\": path is nonlocal", source)
		}

		if !filepath.IsLocal(destination) {
			return fmt.Errorf("invalid destination resource \"%s\": path is nonlocal", destination)
		}

		log.Printf("[REPLACE] Transferring: %s -> %s", source, destination)

		resourceName := filepath.Clean(destination)
		if !cache.Has(resourceName) {
			resource, err := client.ReadResource(clientDirectory, resourceName)
			if err != nil {
				return fmt.Errorf("could not read patch destination: %v", err)
			}

			log.Printf("Adding %s to cache", resource.Path)
			err = cache.Add(resource)
			if err != nil {
				return fmt.Errorf("could not add patch destination to cache: %v", err)
			}
		}

		sourcePath := filepath.Join("patches", server.Id, patch.Version, source)
		data, err := os.ReadFile(sourcePath)
		if err != nil {
			return fmt.Errorf("could not read patch source: %v", err)
		}

		destinationPath := filepath.Join(clientDirectory, resourceName)
		err = os.WriteFile(destinationPath, data, 0755)
		if err != nil {
			return fmt.Errorf("cannot write patch source: %v", err)
		}
	}

	return nil
}

func (patch *tppPatch) addResources(clientDirectory string, cache client.Cache, server *server.Server) error {
	for source, destination := range patch.Add {
		if !filepath.IsLocal(source) {
			return fmt.Errorf("invalid source resource \"%s\": path is nonlocal", source)
		}

		if !filepath.IsLocal(destination) {
			return fmt.Errorf("invalid destination resource \"%s\": path is nonlocal", destination)
		}

		log.Printf("[ADD] Transferring: %s -> %s", source, destination)

		if client.Contains(clientDirectory, destination) {
			return fmt.Errorf("cannot tranfer \"%s\" to \"%s\": resource already exists", source, destination)
		}

		sourcePath := filepath.Join("patches", server.Id, patch.Version, source)
		data, err := os.ReadFile(sourcePath)
		if err != nil {
			return fmt.Errorf("cannot read patch source: %v", err)
		}

		destinationName := filepath.Join(clientDirectory, destination)
		err = os.WriteFile(destinationName, data, 0755)
		if err != nil {
			return fmt.Errorf("cannot write patch source: %v", err)
		}
	}

	return nil
}

func (patch *tppPatch) doTransfer(clientDirectory string, cache client.Cache, server *server.Server) error {
	return errors.Join(
		patch.replaceResources(clientDirectory, cache, server),
		patch.addResources(clientDirectory, cache, server),
	)
}

func (patch *tppPatch) TransferResources(clientDirectory string, cache client.Cache, server *server.Server) error {
	dependencies, err := patch.getDependencies(server)
	if err != nil {
		return err
	}

	for _, dependency := range dependencies {
		err := dependency.doTransfer(clientDirectory, cache, server)
		if err != nil {
			return fmt.Errorf("transfer dependency: %v", err)
		}
	}

	return patch.TransferResources(clientDirectory, cache, server)
}

var _ Patch = &tppPatch{}
