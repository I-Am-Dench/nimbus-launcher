package patch

import (
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
)

var _ Patch = &Tpp{}

type Dependent interface {
	GetDependencies(server Server, recursive ...bool) ([]Patch, error)
}

type Tpp struct {
	version string `json:"-"`

	Dependencies []string `json:"depend,omitempty"`

	Download map[string]string `json:"download,omitempty"`
	// Download []struct {
	// 	Path string `json:"path"`
	// 	Name string `json:"name"`
	// } `json:"download,omitempty"`

	Update struct {
		Boot     string `json:"boot,omitempty"`
		Protocol string `json:"protocol,omitempty"`
	} `json:"update,omitempty"`

	Replace map[string]string `json:"replace,omitempty"`
	Add     map[string]string `json:"add,omitempty"`
}

func NewTpp(version string) Patch {
	return &Tpp{version: version}
}

func (patch *Tpp) Version() string {
	return patch.version
}

func (patch *Tpp) doDownloads(server Server) error {
	log.Println("Starting downloads...")
	downloadPath := filepath.Join(server.DownloadDir(), patch.version)
	os.MkdirAll(downloadPath, 0755)

	for path, name := range patch.Download {
		if !filepath.IsLocal(name) {
			return &PatchError{fmt.Errorf("invalid download name \"%s\": name is nonlocal", name)}
		}

		response, err := server.RemoteGet(path)
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

		file, err := os.OpenFile(filepath.Join(downloadPath, name), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
		if err != nil {
			return &PatchError{fmt.Errorf("could not open file in download directory: %v", err)}
		}
		defer file.Close()

		_, err = io.Copy(file, response.Body)
		if err != nil {
			return &PatchError{fmt.Errorf("could not save download \"%s\" to \"%s\": %v", path, name, err)}
		}
	}

	return nil
}

func (patch *Tpp) DownloadResources(server Server, rejections *RejectionList) error {
	if rejections.IsRejected(server, patch.version) {
		return &PatchError{fmt.Errorf("\"%s\" is rejected", patch.version)}
	}

	if err := ValidateVersionName(patch.version); err != nil {
		return &PatchError{err}
	}

	return patch.doDownloads(server)
}

func (patch *Tpp) parseDependencyVersion(version string) (string, bool) {
	trimmed := strings.TrimSpace(version)
	if len(trimmed) > 0 && trimmed[len(trimmed)-1] == '*' {
		return trimmed[:len(trimmed)-1], true
	}
	return trimmed, false
}

func (patch *Tpp) GetDependencies(server Server, recursive ...bool) ([]Patch, error) {
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

		dependency, err := server.GetPatch(version)
		if err != nil {
			return []Patch{}, fmt.Errorf("cannot resolve patch dependency \"%s\": %v", version, err)
		}

		patches = append(patches, dependency)

		if dependent, ok := dependency.(Dependent); ok && (fetchSubDependencies || recurse) {
			subDependencies, err := dependent.GetDependencies(server, recurse)
			if err != nil {
				return []Patch{}, fmt.Errorf("cannot resolve recursive dependency \"%s\": %v", version, err)
			}
			patches = append(patches, subDependencies...)
		}
	}

	return patches, nil
}

func (patch *Tpp) updateBoot(server Server) error {
	if len(patch.Update.Boot) == 0 {
		return nil
	}

	log.Println("Updating boot file...")
	path := filepath.Join(server.DownloadDir(), patch.version)

	data, err := os.ReadFile(filepath.Join(path, patch.Update.Boot))
	if err != nil {
		return fmt.Errorf("could not read boot patch file \"%s\": %v", patch.Update.Boot, err)
	}

	config := &ldf.BootConfig{}
	err = ldf.Unmarshal(data, config)
	if err != nil {
		return fmt.Errorf("could not unmarshal boot patch file: %v", err)
	}

	return server.SetBootConfig(config)
}

func (patch *Tpp) updateProtocol(server Server) error {
	if len(patch.Update.Protocol) == 0 {
		return nil
	}

	log.Printf("Updating patch server protocol to \"%s\"", patch.Update.Protocol)
	server.SetPatchProtocol(patch.Update.Protocol)

	return nil
}

func (patch *Tpp) doUpdates(server Server) error {
	return errors.Join(
		patch.updateBoot(server),
		patch.updateProtocol(server),
	)
}

func (patch *Tpp) UpdateResources(server Server, rejections *RejectionList) error {
	dependencies, err := patch.GetDependencies(server)
	if err != nil {
		return &PatchError{err}
	}

	for _, dependency := range dependencies {
		err := dependency.DownloadResources(server, rejections)
		if err != nil {
			return &PatchError{fmt.Errorf("dependency \"%s\": %v", dependency.Version(), err)}
		}
	}

	err = patch.DownloadResources(server, rejections)
	if err != nil {
		return &PatchError{err}
	}

	return patch.doUpdates(server)
}

func (patch *Tpp) replace(source, destination string) error {
	sourceFile, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("could not open patch source: %v", err)
	}
	defer sourceFile.Close()

	destinationFile, err := os.OpenFile(destination, os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return fmt.Errorf("could not open patch destination: %v", err)
	}
	defer destinationFile.Close()

	_, err = io.Copy(destinationFile, sourceFile)
	if err != nil {
		return fmt.Errorf("could not copy \"%s\" to \"%s\": %v", source, destination, err)
	}

	return nil
}

func (patch *Tpp) replaceResources(clientDirectory string, cache client.Cache, server Server) error {
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

		sourcePath := filepath.Join(server.DownloadDir(), patch.version, source)
		destinationPath := filepath.Join(clientDirectory, resourceName)

		err := patch.replace(sourcePath, destinationPath)
		if err != nil {
			return err
		}
	}

	return nil
}

func (patch *Tpp) add(source, destination string) error {
	sourceFile, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("could not open patch source: %v", err)
	}
	defer sourceFile.Close()

	destinationFile, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return fmt.Errorf("could not open patch destination: %v", err)
	}
	defer destinationFile.Close()

	_, err = io.Copy(destinationFile, sourceFile)
	if err != nil {
		return fmt.Errorf("could not copy \"%s\" to \"%s\": %v", source, destination, err)
	}

	return nil
}

func (patch *Tpp) addResources(clientDirectory string, server Server) error {
	for source, destination := range patch.Add {
		if !filepath.IsLocal(source) {
			return fmt.Errorf("invalid source resource \"%s\": path is nonlocal", source)
		}

		if !filepath.IsLocal(destination) {
			return fmt.Errorf("invalid destination resource \"%s\": path is nonlocal", destination)
		}

		log.Printf("[ADD] Transferring: %s -> %s", source, destination)

		if client.Contains(clientDirectory, destination) {
			return fmt.Errorf("cannot transfer \"%s\" to \"%s\": resource already exists", source, destination)
		}

		sourcePath := filepath.Join(server.DownloadDir(), patch.version, source)
		destinationPath := filepath.Join(clientDirectory, destination)

		err := patch.add(sourcePath, destinationPath)
		if err != nil {
			return err
		}
	}

	return nil
}

func (patch *Tpp) TransferResources(clientDirectory string, cache client.Cache, server Server) error {
	return errors.Join(
		patch.replaceResources(clientDirectory, cache, server),
		patch.addResources(clientDirectory, server),
	)
}

func (patch *Tpp) TransferResourcesWithDependencies(clientDirectory string, cache client.Cache, server Server) error {
	dependencies, err := patch.GetDependencies(server)
	if err != nil {
		return err
	}

	for _, dependency := range dependencies {
		err := dependency.TransferResources(clientDirectory, cache, server)
		if err != nil {
			return fmt.Errorf("transfer dependecy: %v", err)
		}
	}

	return patch.TransferResources(clientDirectory, cache, server)
}

func (patch *Tpp) Summary() string {
	updates := 0

	if len(patch.Update.Boot) > 0 {
		updates++
	}

	if len(patch.Update.Protocol) > 0 {
		updates++
	}

	return fmt.Sprintf("%d download(s); %d update(s); %d replacement(s); %d addition(s)", len(patch.Download), updates, len(patch.Replace), len(patch.Add))
}
