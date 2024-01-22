package patch_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/I-Am-Dench/lu-launcher/client"
	"github.com/I-Am-Dench/lu-launcher/resource/patch"
	"github.com/I-Am-Dench/lu-launcher/resource/server"
)

type cache struct {
	m map[string]client.ClientResource
}

func (cache *cache) Add(resource client.ClientResource) error {
	cache.m[resource.Path] = resource
	return nil
}

func (cache *cache) Get(path string) (client.ClientResource, error) {
	resource, ok := cache.m[path]
	if !ok {
		return client.ClientResource{}, fmt.Errorf("cache: \"%s\" does not exist", path)
	}

	return resource, nil
}

func (cache *cache) GetResources() ([]client.ClientResource, error) {
	resources := []client.ClientResource{}
	for _, resource := range cache.m {
		resources = append(resources, resource)
	}
	return resources, nil
}

func (cache *cache) Has(path string) bool {
	_, ok := cache.m[path]
	return ok
}

func (cache *cache) Close() error {
	return nil
}

func readTestPatch(name string) []byte {
	data, err := os.ReadFile(filepath.Join("test_patches", name))
	if err != nil {
		panic(fmt.Errorf("read test patch: %v", err))
	}

	return data
}

var testVersions = server.PatchesSummary{
	CurrentVersion:   "v1.0.0",
	PreviousVersions: []string{},
}

func serverFileSystem() fileSystem {
	fs := make(fileSystem)

	versions, _ := json.Marshal(testVersions)
	fs["/patches"] = versions

	fs["/patches/v1.0.0"] = readTestPatch("patch1.json")

	fs["/patches/v1.0.0/a"] = []byte("Test 1")
	fs["/patches/v1.0.0/b"] = []byte("Test 2")
	fs["/patches/v1.0.0/c"] = []byte("Test 3")

	return fs
}

func clientFileSystem() fileSystem {
	fs := make(fileSystem)

	fs["data/file1"] = []byte("default data 1")
	fs["data/file2"] = []byte("default data 2")
	fs["data/file3"] = []byte("default data 3")

	return fs
}

func expectedClient() fileSystem {
	fs := make(fileSystem)

	fs["data/file1"] = []byte("Test 1")
	fs["data/file2"] = []byte("Test 2")
	fs["data/file3"] = []byte("Test 3")

	return fs
}

func hasSameContent(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}

	for i, v := range a {
		if v != b[i] {
			return false
		}
	}

	return true
}

func checkContents(dir, path string, expected []byte) error {
	actual, err := os.ReadFile(filepath.Join(dir, path))
	if err != nil {
		return fmt.Errorf("check contents: %v", err)
	}

	if !hasSameContent(expected, actual) {
		return fmt.Errorf("expected `%v` but read `%v`", expected, actual)
	}

	return nil
}

func TestPatching(t *testing.T) {
	serverFS := serverFileSystem()
	clientFS := clientFileSystem()

	env, teardown := setup(t, serverFS)
	defer teardown()

	clientDirectory := filepath.Join(env.Dir, "client")

	rejections := patch.NewRejectionList(filepath.Join(env.Dir, "rejections.json"))
	clientCache := &cache{
		m: make(map[string]client.ClientResource),
	}

	_, err := env.ServerConfig.GetPatch("v1.0.0")
	if !errors.Is(err, patch.ErrPatchesUnavailable) {
		t.Fatalf("test patching: Server.GetPatch did not return patch.ErrPatchesUnavailable: instead: %v", err)
	}

	go env.PatchServer.ListenAndServe()

	t.Log("Started test patch server.")

	clientFS.Init(clientDirectory, t)

	patch, err := env.ServerConfig.GetPatch("v1.0.0")
	if err != nil {
		t.Fatalf("test patching: %v", err)
	}

	err = patch.UpdateResources(env.ServerConfig, rejections)
	if err != nil {
		t.Fatalf("test patching: update resources: %v", err)
	}

	err = patch.TransferResources(clientDirectory, clientCache, env.ServerConfig)
	if err != nil {
		t.Fatalf("test patching: transfer resources: %v", err)
	}

	expected := expectedClient()

	for path, expectedData := range expected {
		err := checkContents(clientDirectory, path, expectedData)
		if err != nil {
			t.Errorf("test patching: %v", err)
		} else {
			t.Logf("\"%s\" is correct!", path)
		}
	}
}
