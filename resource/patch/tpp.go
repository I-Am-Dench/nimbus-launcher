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

func NewTpp(version string) Patch {
	return &Tpp{version: version}
}

func (patch *Tpp) Version() string {
	return patch.version
}

func (patch *Tpp) doDownloads(server Server) error {
	log.Println("Starting downloads...")
	path := filepath.Join(server.DownloadDir(), patch.version)
	os.MkdirAll(path, 0755)

	for _, download := range patch.Download {
		if !filepath.IsLocal(download.Name) {
			return &PatchError{fmt.Errorf("invalid download name \"%s\": name is nonlocal", download.Name)}
		}

		response, err := server.RemoteGet(download.Path)
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

		file, err := os.OpenFile(filepath.Join(path, download.Name), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
		if err != nil {
			return &PatchError{fmt.Errorf("could not open file in download directory: %v", err)}
		}
		defer file.Close()

		_, err = io.Copy(file, response.Body)
		if err != nil {
			return &PatchError{fmt.Errorf("could not save download \"%s\" to \"%s\": %v", download.Path, download.Name, err)}
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

// type Tpp struct {
// 	Version string `json:"-"`
// 	dir     string `json:"-"`

// 	Dependencies []string `json:"depend,omitempty"`

// 	Download []struct {
// 		Path string `json:"path"`
// 		Name string `json:"name"`
// 	} `json:"download,omitempty"`

// 	Update struct {
// 		Boot     string `json:"boot,omitempty"`
// 		Protocol string `json:"protocol,omitempty"`
// 	} `json:"update,omitempty"`

// 	Replace map[string]string `json:"replace,omitempty"`
// 	Add     map[string]string `json:"add,omitempty"`
// }

// func NewTpp(localDir, version string) *Tpp {
// 	return &Tpp{Version: version, dir: localDir}
// }

// func (p *Tpp) getPatch(version string, server *server.Server) (*Tpp, error) {
// 	patchDirectory := filepath.Join(p.dir, server.Id, version)
// 	path := filepath.Join(patchDirectory, "patch.json")

// 	data, err := os.ReadFile(path)
// 	if err == nil {
// 		patch := NewTpp(p.dir, version)

// 		err = json.Unmarshal(data, &patch)
// 		if err != nil {
// 			return nil, fmt.Errorf("cannot unmarshal \"%s\": %v", path, err)
// 		}

// 		return patch, nil
// 	}

// 	response, err := server.RemoteGet(version)
// 	if err != nil {
// 		return nil, ErrPatchesUnavailable
// 	}
// 	defer response.Body.Close()

// 	if response.StatusCode == http.StatusUnauthorized {
// 		return nil, ErrPatchesUnauthorized
// 	}

// 	if response.StatusCode >= 400 {
// 		return nil, ErrPatchesUnavailable
// 	}

// 	data, err = io.ReadAll(response.Body)
// 	if err != nil {
// 		return nil, fmt.Errorf("cannot read body of patch version response: %v", err)
// 	}

// 	patch := NewTpp(p.dir, version)
// 	err = json.Unmarshal(data, &patch)
// 	if err != nil {
// 		return nil, fmt.Errorf("malformed response body from patch version: %v", err)
// 	}

// 	os.MkdirAll(patchDirectory, 0755)

// 	err = os.WriteFile(path, data, 0755)
// 	if err != nil {
// 		log.Printf("Could not save patch.json: %v", err)
// 	}

// 	return patch, err
// }

// func (patch *Tpp) doDownload(server *server.Server) error {
// 	log.Println("Starting downloads...")
// 	path := filepath.Join(patch.dir, server.Id, patch.Version)
// 	os.MkdirAll(path, 0755)

// 	for _, download := range patch.Download {
// 		if !filepath.IsLocal(download.Name) {
// 			return &PatchError{fmt.Errorf("invalid download name \"%s\": name is nonlocal", download.Name)}
// 		}

// 		response, err := server.RemoteGet(download.Name)
// 		if err != nil {
// 			return &PatchError{fmt.Errorf("could not get url: %v", err)}
// 		}
// 		defer response.Body.Close()

// 		if response.StatusCode == http.StatusUnauthorized {
// 			return &PatchError{ErrPatchesUnauthorized}
// 		}

// 		if response.StatusCode >= 400 {
// 			return &PatchError{fmt.Errorf("invalid response status code from server: %d", response.StatusCode)}
// 		}

// 		data, err := io.ReadAll(response.Body)
// 		if err != nil {
// 			return &PatchError{fmt.Errorf("could not read body of server response: %v", err)}
// 		}

// 		err = os.WriteFile(filepath.Join(path, download.Name), data, 0755)
// 		if err != nil {
// 			return &PatchError{fmt.Errorf("could not save download \"%s\" to \"%s\": %v", download.Path, download.Name, err)}
// 		}
// 	}

// 	return nil
// }

// func (patch *Tpp) DownloadResources(server *server.Server, rejections RejectionList) error {
// 	if rejections.IsRejected(server, patch.Version) {
// 		return &PatchError{fmt.Errorf("\"%s\" is rejected", patch.Version)}
// 	}

// 	if err := ValidateVersionName(patch.Version); err != nil {
// 		return &PatchError{fmt.Errorf("%v", err)}
// 	}

// 	return patch.doDownload(server)
// }

// func (patch *Tpp) parseDependencyVersion(version string) (string, bool) {
// 	trimmed := strings.TrimSpace(version)
// 	if len(trimmed) > 0 && trimmed[len(trimmed)-1] == '*' {
// 		return trimmed[:len(trimmed)-1], true
// 	}
// 	return trimmed, false
// }

// func (patch *Tpp) getDependencies(server *server.Server, recursive ...bool) ([]*Tpp, error) {
// 	recurse := false
// 	if len(recursive) > 0 {
// 		recurse = recursive[0]
// 	}

// 	patches := []*Tpp{}

// 	for _, dependencyName := range patch.Dependencies {
// 		version, fetchSubDependencies := patch.parseDependencyVersion(dependencyName)
// 		if len(version) == 0 {
// 			continue
// 		}

// 		dependency, err := patch.getPatch(version, server)
// 		if err != nil {
// 			return []*Tpp{}, fmt.Errorf("cannot resolve patch dependency \"%s\": %v", version, err)
// 		}

// 		patches = append(patches, dependency)
// 		if fetchSubDependencies || recurse {
// 			subDependencies, err := dependency.getDependencies(server, recurse)
// 			if err != nil {
// 				return []*Tpp{}, fmt.Errorf("cannot resolve recursive dependency \"%s\": %v", version, err)
// 			}
// 			patches = append(patches, subDependencies...)
// 		}
// 	}

// 	return patches, nil
// }

// func (patch *Tpp) updateBoot(server *server.Server) error {
// 	if len(patch.Update.Boot) == 0 {
// 		return nil
// 	}

// 	log.Println("Updating boot file...")
// 	path := filepath.Join(patch.dir, server.Id, patch.Version)

// 	data, err := os.ReadFile(filepath.Join(path, patch.Update.Boot))
// 	if err != nil {
// 		return fmt.Errorf("could not read boot patch file \"%s\": %v", patch.Update.Boot, err)
// 	}

// 	config := &ldf.BootConfig{}
// 	err = ldf.Unmarshal(data, config)
// 	if err != nil {
// 		return fmt.Errorf("could not unmarshal boot patch file: %v", err)
// 	}

// 	server.Config = config
// 	return server.SaveConfig()
// }

// func (patch *Tpp) updateProtocol(server *server.Server) error {
// 	if len(patch.Update.Protocol) == 0 {
// 		return nil
// 	}

// 	log.Printf("Updating patch server protocol to \"%s\"", patch.Update.Protocol)
// 	server.PatchProtocol = patch.Update.Protocol

// 	return nil
// }

// func (patch *Tpp) doUpdates(server *server.Server) error {
// 	return errors.Join(
// 		patch.updateBoot(server),
// 		patch.updateProtocol(server),
// 	)
// }

// func (patch *Tpp) UpdateResources(server *server.Server, rejections RejectionList) error {
// 	dependencies, err := patch.getDependencies(server)
// 	if err != nil {
// 		return err
// 	}

// 	for _, dependency := range dependencies {
// 		err := dependency.DownloadResources(server, rejections)
// 		if err != nil {
// 			return &PatchError{fmt.Errorf("run dependency \"%s\": %v", dependency.Version, err)}
// 		}
// 	}

// 	err = patch.DownloadResources(server, rejections)
// 	if err != nil {
// 		return &PatchError{err}
// 	}

// 	return patch.doUpdates(server)
// }

// func (patch *Tpp) replaceResources(clientDirectory string, cache client.Cache, server *server.Server) error {
// 	for source, destination := range patch.Replace {
// 		if !filepath.IsLocal(source) {
// 			return fmt.Errorf("invalid source resource \"%s\": path is nonlocal", source)
// 		}

// 		if !filepath.IsLocal(destination) {
// 			return fmt.Errorf("invalid destination resource \"%s\": path is nonlocal", destination)
// 		}

// 		log.Printf("[REPLACE] Transferring: %s -> %s", source, destination)

// 		resourceName := filepath.Clean(destination)
// 		if !cache.Has(resourceName) {
// 			resource, err := client.ReadResource(clientDirectory, resourceName)
// 			if err != nil {
// 				return fmt.Errorf("could not read patch destination: %v", err)
// 			}

// 			log.Printf("Adding %s to cache", resource.Path)
// 			err = cache.Add(resource)
// 			if err != nil {
// 				return fmt.Errorf("could not add patch destination to cache: %v", err)
// 			}
// 		}

// 		sourcePath := filepath.Join("patches", server.Id, patch.Version, source)
// 		data, err := os.ReadFile(sourcePath)
// 		if err != nil {
// 			return fmt.Errorf("could not read patch source: %v", err)
// 		}

// 		destinationPath := filepath.Join(clientDirectory, resourceName)
// 		err = os.WriteFile(destinationPath, data, 0755)
// 		if err != nil {
// 			return fmt.Errorf("cannot write patch source: %v", err)
// 		}
// 	}

// 	return nil
// }

// func (patch *Tpp) addResources(clientDirectory string, cache client.Cache, server *server.Server) error {
// 	for source, destination := range patch.Add {
// 		if !filepath.IsLocal(source) {
// 			return fmt.Errorf("invalid source resource \"%s\": path is nonlocal", source)
// 		}

// 		if !filepath.IsLocal(destination) {
// 			return fmt.Errorf("invalid destination resource \"%s\": path is nonlocal", destination)
// 		}

// 		log.Printf("[ADD] Transferring: %s -> %s", source, destination)

// 		if client.Contains(clientDirectory, destination) {
// 			return fmt.Errorf("cannot transfer \"%s\" to \"%s\": resource already exists", source, destination)
// 		}

// 		sourcePath := filepath.Join("patches", server.Id, patch.Version, source)
// 		data, err := os.ReadFile(sourcePath)
// 		if err != nil {
// 			return fmt.Errorf("cannot read patch source: %v", err)
// 		}

// 		destinationName := filepath.Join(clientDirectory, destination)
// 		err = os.WriteFile(destinationName, data, 0755)
// 		if err != nil {
// 			return fmt.Errorf("cannot write patch source: %v", err)
// 		}
// 	}

// 	return nil
// }

// func (patch *Tpp) doTransfer(clientDirectory string, cache client.Cache, server *server.Server) error {
// 	return errors.Join(
// 		patch.replaceResources(clientDirectory, cache, server),
// 		patch.addResources(clientDirectory, cache, server),
// 	)
// }

// func (patch *Tpp) TransferResources(clientDirectory string, cache client.Cache, server *server.Server) error {
// 	dependencies, err := patch.getDependencies(server)
// 	if err != nil {
// 		return err
// 	}

// 	for _, dependency := range dependencies {
// 		err := dependency.doTransfer(clientDirectory, cache, server)
// 		if err != nil {
// 			return fmt.Errorf("transfer dependency: %v", err)
// 		}
// 	}

// 	return patch.TransferResources(clientDirectory, cache, server)
// }
