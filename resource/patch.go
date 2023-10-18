package resource

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
	"github.com/I-Am-Dench/lu-launcher/luconfig"
)

var (
	ErrPatchesUnsupported  = errors.New("patches unsupoorted")
	ErrPatchesUnavailable  = errors.New("patch server could not be reached")
	ErrPatchesUnauthorized = errors.New("invalid patch token")
)

type PatchVersions struct {
	CurrentVersion   string   `json:"currentVersion"`
	PreviousVersions []string `json:"previousVersions"`
}

func GetPatchVersions(server *Server) (PatchVersions, error) {
	response, err := server.PatchServerRequest()
	if err != nil {
		return PatchVersions{}, ErrPatchesUnavailable
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusServiceUnavailable {
		return PatchVersions{}, ErrPatchesUnsupported
	}

	if response.StatusCode >= 400 {
		return PatchVersions{}, fmt.Errorf("invalid response status code from patch server (%d)", response.StatusCode)
	}

	data, err := io.ReadAll(response.Body)
	if err != nil {
		return PatchVersions{}, fmt.Errorf("cannot read body of patch server response: %v", err)
	}

	patches := PatchVersions{}
	err = json.Unmarshal(data, &patches)
	if err != nil {
		return PatchVersions{}, fmt.Errorf("malformed response body from patch server: %v", err)
	}

	return patches, nil
}

type Patch struct {
	Version string `json:"-"`

	Dependencies []string `json:"depend,omitempty"`

	Download []struct {
		Path string `json:"path"`
		Name string `json:"name"`
	} `json:"download,omitempty"`

	Update struct {
		Boot     string `json:"boot,omitempty"`
		Protocol string `json:"protocol,omitempty"`
	} `json:"update,omitempty"`

	Transfer map[string]string `json:"transfer,omitempty"`
}

func GetPatch(version string, server *Server) (Patch, error) {
	patchDirectory := filepath.Join("patches", server.Id, version)
	path := filepath.Join(patchDirectory, "patch.json")

	data, err := os.ReadFile(path)
	if err == nil {
		patch := Patch{
			Version: version,
		}
		err = json.Unmarshal(data, &patch)
		if err != nil {
			return Patch{}, fmt.Errorf("cannot unmarshal \"%s\": %v", path, err)
		}
		return patch, err
	}

	response, err := server.PatchServerRequest(version)
	if err != nil {
		return Patch{}, ErrPatchesUnavailable
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusUnauthorized {
		return Patch{}, ErrPatchesUnauthorized
	}

	if response.StatusCode >= 400 {
		return Patch{}, ErrPatchesUnavailable
	}

	data, err = io.ReadAll(response.Body)
	if err != nil {
		return Patch{}, fmt.Errorf("cannot read body of patch version response: %v", err)
	}

	patch := Patch{
		Version: version,
	}
	err = json.Unmarshal(data, &patch)
	if err != nil {
		return Patch{}, fmt.Errorf("malformed response body from patch version: %v", err)
	}

	os.MkdirAll(patchDirectory, 0755)

	err = os.WriteFile(path, data, 0755)
	if err != nil {
		log.Printf("Could not save patch.json: %v", err)
	}

	return patch, nil
}

func (patch *Patch) doDownload(server *Server) error {
	log.Println("Starting downloads...")
	path := filepath.Join("patches", server.Id, patch.Version)
	os.MkdirAll(path, 0755)

	for _, download := range patch.Download {
		if !filepath.IsLocal(download.Name) {
			return fmt.Errorf("invalid download name \"%s\": name is a nonlocal path", download.Name)
		}

		response, err := server.PatchServerRequest(download.Path)
		if err != nil {
			return fmt.Errorf("could not GET download URL: %v", err)
		}
		defer response.Body.Close()

		if response.StatusCode == http.StatusUnauthorized {
			return ErrPatchesUnauthorized
		}

		if response.StatusCode >= 400 {
			return fmt.Errorf("invalid response status code patch server: %d", response.StatusCode)
		}

		data, err := io.ReadAll(response.Body)
		if err != nil {
			return fmt.Errorf("could not read body of download response: %v", err)
		}

		err = os.WriteFile(filepath.Join(path, download.Name), data, 0755)
		if err != nil {
			return fmt.Errorf("could not save download \"%s\" to \"%s\": %v", download.Path, download.Name, err)
		}
	}

	return nil
}

func (patch *Patch) updateBoot(server *Server) error {
	if len(patch.Update.Boot) == 0 {
		return nil
	}

	log.Println("Updating boot file...")
	path := filepath.Join("patches", server.Id, patch.Version)

	data, err := os.ReadFile(filepath.Join(path, patch.Update.Boot))
	if err != nil {
		return fmt.Errorf("could not read boot patch file file \"%s\": %v", patch.Update.Boot, err)
	}

	config := luconfig.New()
	err = luconfig.Unmarshal(data, config)
	if err != nil {
		return fmt.Errorf("could not unmarshal boot patch file: %v", err)
	}

	server.Config = config
	return server.SaveConfig()
}

func (patch *Patch) updateProtocol(server *Server) error {
	if len(patch.Update.Protocol) == 0 {
		return nil
	}

	log.Printf("Updating patch server protocol to \"%s\"\n", patch.Update.Protocol)
	server.PatchProtocol = patch.Update.Protocol

	return nil
}

func (patch *Patch) doUpdates(server *Server) error {
	return errors.Join(
		patch.updateBoot(server),
		patch.updateProtocol(server),
	)
}

func (patch *Patch) parseDependencyVersion(version string) (string, bool) {
	stripped := strings.TrimSpace(version)
	if len(stripped) > 0 && stripped[len(stripped)-1] == '*' {
		return stripped[:len(stripped)-1], true
	}
	return stripped, false
}

func (patch *Patch) getDependencies(server *Server, recursive ...bool) ([]Patch, error) {
	recurse := false
	if len(recursive) > 0 {
		recurse = recursive[0]
	}

	patches := []Patch{}

	for _, dependencyName := range patch.Dependencies {
		version, fetchSubDependencies := patch.parseDependencyVersion(dependencyName)
		if len(version) == 0 {
			continue
		}

		dependency, err := GetPatch(version, server)
		if err != nil {
			return []Patch{}, fmt.Errorf("cannot resolve patch dependency '%s': %v", version, err)
		}

		patches = append(patches, dependency)
		if fetchSubDependencies || recurse {
			subDependencies, err := dependency.getDependencies(server, recurse)
			if err != nil {
				return []Patch{}, fmt.Errorf("cannot resolve recursive dependency '%s': %v", version, err)
			}
			patches = append(patches, subDependencies...)
		}
	}

	return patches, nil
}

func (patch *Patch) Run(server *Server, rejected RejectedPatches) error {
	if rejected.IsRejected(server, patch.Version) {
		return fmt.Errorf("run patch: \"%s\" is rejected", patch.Version)
	}

	if err := ValidateVersionName(patch.Version); err != nil {
		return fmt.Errorf("run patch: %v", err)
	}

	return patch.doDownload(server)
}

func (patch *Patch) RunWithDependencies(server *Server, rejected RejectedPatches) error {
	dependencies, err := patch.getDependencies(server)
	if err != nil {
		return err
	}

	for _, dependency := range dependencies {
		err := dependency.Run(server, rejected)
		if err != nil {
			return fmt.Errorf("run dependency \"%s\": %v", dependency.Version, err)
		}
	}

	err = patch.Run(server, rejected)
	if err != nil {
		return fmt.Errorf("run patch \"%s\": %v", patch.Version, err)
	}

	return patch.doUpdates(server)
}

func (patch *Patch) TransferResources(clientDirectory string, cache client.Cache, server *Server) error {
	for source, destination := range patch.Transfer {
		if !filepath.IsLocal(source) {
			return fmt.Errorf("invalid source resource \"%s\": file is nonlocal", source)
		}

		if !filepath.IsLocal(destination) {
			return fmt.Errorf("invalid destination resource \"%s\": file is nonlocal", destination)
		}

		log.Printf("Transferring: %s -> %s\n", source, destination)

		resourceName := filepath.Clean(destination)
		if !cache.Has(resourceName) {
			resource, err := client.ReadResource(clientDirectory, resourceName)
			if err != nil {
				return fmt.Errorf("could not read patch destination: %v", err)
			}

			log.Printf("Adding %s to cache\n", resource.Path)
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

func (patch *Patch) TransferAllResources(clientDirectory string, cache client.Cache, server *Server) error {
	dependencies, err := patch.getDependencies(server)
	if err != nil {
		return err
	}

	for _, dependency := range dependencies {
		err := dependency.TransferResources(clientDirectory, cache, server)
		if err != nil {
			return fmt.Errorf("transfer resources: %v", err)
		}
	}

	return patch.TransferResources(clientDirectory, cache, server)
}
