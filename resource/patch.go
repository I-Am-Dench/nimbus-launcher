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
	ErrPatchesUnsupported  = errors.New("patches unsupported")
	ErrPatchesUnavailable  = errors.New("patch server could not be reached")
	ErrPatchesUnauthorized = errors.New("invalid patch token")
)

type ServerPatches struct {
	CurrentVersion string   `json:"currentVersion"`
	Patches        []string `json:"versions"`
}

func GetServerPatches(server *Server) (ServerPatches, error) {
	response, err := server.PatchServerRequest()
	if err != nil {
		return ServerPatches{}, ErrPatchesUnavailable
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusServiceUnavailable {
		return ServerPatches{}, ErrPatchesUnsupported
	}

	if response.StatusCode == http.StatusUnauthorized {
		return ServerPatches{}, ErrPatchesUnauthorized
	}

	if response.StatusCode >= 300 {
		return ServerPatches{}, fmt.Errorf("invalid response status code from patch server (%d)", response.StatusCode)
	}

	data, err := io.ReadAll(response.Body)
	if err != nil {
		return ServerPatches{}, fmt.Errorf("cannot read body of patch server response: %v", err)
	}

	patches := ServerPatches{}
	err = json.Unmarshal(data, &patches)
	if err != nil {
		return ServerPatches{}, fmt.Errorf("malformed response body from patch server: %v", err)
	}

	return patches, nil
}

type Patch struct {
	Version string `json:"-"`

	Dependencies []string `json:"depend,omitempty"`

	Downloads []struct {
		Path string `json:"path"`
		Name string `json:"name"`
	} `json:"downloads,omitempty"`

	Updates struct {
		Boot string `json:"boot,omitempty"`
	} `json:"updates,omitempty"`

	Transfers map[string]string `json:"transfer,omitempty"`
}

func GetPatch(version string, server *Server) (Patch, error) {
	path := filepath.Join("patches", server.Id, version)

	data, err := os.ReadFile(filepath.Join(path, "patch.json"))
	if err == nil {
		patch := Patch{
			Version: version,
		}
		err = json.Unmarshal(data, &patch)
		if err != nil {
			return Patch{}, fmt.Errorf("cannot unmarshal \"%s/patch.json\": %v", path, err)
		}
		return patch, nil
	}

	response, err := server.PatchServerRequest(version)
	if err != nil {
		return Patch{}, ErrPatchesUnavailable
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusUnauthorized {
		return Patch{}, ErrPatchesUnauthorized
	}

	if response.StatusCode >= 300 {
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
		return Patch{}, fmt.Errorf("malformed response body for patch version: %v", err)
	}

	os.MkdirAll(path, 0755)

	err = os.WriteFile(filepath.Join(path, "patch.json"), data, 0755)
	if err != nil {
		log.Printf("Could not save patch.json: %v", err)
	}

	return patch, nil
}

func (patch *Patch) downloadFiles(server *Server) error {
	log.Println("Starting downloads...")
	path := filepath.Join("patches", server.Id, patch.Version)
	os.MkdirAll(path, 0755)

	for _, download := range patch.Downloads {
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

		if response.StatusCode >= 300 {
			return fmt.Errorf("invalid response status code from patch server: %d", response.StatusCode)
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
	if len(patch.Updates.Boot) == 0 {
		return nil
	}

	log.Println("Updating boot file...")
	path := filepath.Join("patches", server.Id, patch.Version)

	data, err := os.ReadFile(filepath.Join(path, patch.Updates.Boot))
	if err != nil {
		return fmt.Errorf("could not read boot patch file \"%s\": %v", patch.Updates.Boot, err)
	}

	config := luconfig.New()
	err = luconfig.Unmarshal(data, config)
	if err != nil {
		return fmt.Errorf("could not unmarshal boot patch file: %v", err)
	}

	server.Config = config
	return server.SaveConfig()
}

func (patch *Patch) updateFiles(server *Server) error {
	return patch.updateBoot(server)
}

func (patch *Patch) parseDependencyVersion(version string) (string, bool) {
	stripped := strings.TrimSpace(version)
	if len(stripped) > 0 && stripped[len(stripped)-1] == '*' {
		return stripped[:len(stripped)-1], true
	}

	return stripped, false
}

func (patch *Patch) getDependencies(server *Server, recurse ...bool) ([]Patch, error) {
	recursive := false
	if len(recurse) > 0 {
		recursive = recurse[0]
	}

	patches := []Patch{}

	for _, dependencyVersion := range patch.Dependencies {
		version, fetchSubDependencies := patch.parseDependencyVersion(dependencyVersion)
		if len(version) == 0 {
			continue
		}

		dependency, err := GetPatch(version, server)
		if err != nil {
			return []Patch{}, fmt.Errorf("cannot resolve patch dependency '%s': %v", version, err)
		}

		patches = append(patches, dependency)
		if fetchSubDependencies || recursive {
			subDependencies, err := dependency.getDependencies(server, recurse...)
			if err != nil {
				return []Patch{}, fmt.Errorf("resolve resursive dependency '%s': %v", version, err)
			}
			patches = append(patches, subDependencies...)
		}
	}

	return patches, nil
}

func (patch *Patch) Run(server *Server) error {
	if err := ValidateVersionName(patch.Version); err != nil {
		return fmt.Errorf("run patch: %v", err)
	}

	return errors.Join(
		patch.downloadFiles(server),
		patch.updateFiles(server),
	)
}

func (patch *Patch) RunWithDependencies(server *Server) error {
	dependencies, err := patch.getDependencies(server)
	if err != nil {
		return err
	}

	for _, dependency := range dependencies {
		err := dependency.Run(server)
		if err != nil {
			return fmt.Errorf("dependency run error: %v", err)
		}
	}

	return patch.Run(server)
}

func (patch *Patch) TransferResources(clientDirectory string, cache client.Cache, server *Server) error {
	for source, destination := range patch.Transfers {
		if !filepath.IsLocal(source) {
			return fmt.Errorf("invalid source resource \"%s\": file is nonlocal", source)
		}

		if !filepath.IsLocal(destination) {
			return fmt.Errorf("invalid destination resource \"%s\": file is nonlocal", destination)
		}

		log.Printf("Transferring: %s -> %s\n", source, destination)

		resourceName := filepath.Clean(destination)
		if !cache.ResourceExists(resourceName) {
			resource, err := client.ReadResource(clientDirectory, resourceName)
			if err != nil {
				return fmt.Errorf("could not read patch destination: %v", err)
			}

			fmt.Printf("Adding %s to cache\n", resource.Path)
			err = cache.Add(resource)
			if err != nil {
				return fmt.Errorf("could not add patch destination to cache: %v", err)
			}
		}

		sourcePath := filepath.Join("patches", server.Id, patch.Version, source)
		data, err := os.ReadFile(sourcePath)
		if err != nil {
			return fmt.Errorf("cannot read patch source: %v", err)
		}

		destinationPath := filepath.Join(clientDirectory, resourceName)
		err = os.WriteFile(destinationPath, data, 0755)
		if err != nil {
			return fmt.Errorf("cannot write patch source: %v", err)
		}
	}

	return nil
}
