package patch_test

import (
	"errors"
	"fmt"
	"io/fs"
	"math/rand"
	"os"
	"path/filepath"
	"testing"

	"github.com/I-Am-Dench/lu-launcher/client"
	"github.com/I-Am-Dench/lu-launcher/ldf"
	"github.com/I-Am-Dench/lu-launcher/resource/patch"
)

type replacementCache struct {
	m map[string]client.Resource
}

func (cache *replacementCache) Add(resource client.Resource) error {
	cache.m[resource.Path] = resource
	return nil
}

func (cache *replacementCache) Get(path string) (client.Resource, error) {
	resource, ok := cache.m[path]
	if !ok {
		return client.Resource{}, fmt.Errorf("cache: \"%s\" does not exist", path)
	}

	return resource, nil
}

func (cache *replacementCache) List() ([]client.Resource, error) {
	resources := []client.Resource{}
	for _, resource := range cache.m {
		resources = append(resources, resource)
	}
	return resources, nil
}

func (cache *replacementCache) Has(path string) bool {
	_, ok := cache.m[path]
	return ok
}

type additionsCache struct {
	m map[string]struct{}
}

func (cache *additionsCache) Add(path string) error {
	cache.m[path] = struct{}{}
	return nil
}

func (cache *additionsCache) Get(path string) (string, error) {
	_, ok := cache.m[path]
	if ok {
		return path, nil
	}

	return "", nil
}

func (cache *additionsCache) List() ([]string, error) {
	paths := []string{}
	for path := range cache.m {
		paths = append(paths, path)
	}
	return paths, nil
}

func (cache *additionsCache) Has(path string) bool {
	_, ok := cache.m[path]
	return ok
}

type resources struct {
	replacements replacementCache
	additions    additionsCache
}

func (resources *resources) Replacements() client.Cache[client.Resource] {
	return &resources.replacements
}

func (resources *resources) Additions() client.Cache[string] {
	return &resources.additions
}

func (*resources) Close() error {
	return nil
}

func readTestPatch(name string) []byte {
	data, err := os.ReadFile(filepath.Join("test_patches", name))
	if err != nil {
		panic(fmt.Errorf("read test patch: %v", err))
	}

	return data
}

func serverFileSystem(updatedBoot *ldf.BootConfig) fileSystem {
	fs := make(fileSystem)

	fs["/patches/v1.0.0"] = readTestPatch("patch1.json")
	fs["/patches/v2.0.0"] = readTestPatch("patch2.json")
	fs["/patches/v3.0.0"] = readTestPatch("patch3.json")
	fs["/patches/v4.0.0"] = readTestPatch("patch4.json")
	fs["/patches/v5.0.0"] = readTestPatch("patch5.json")
	fs["/patches/v6.0.0"] = readTestPatch("patch6.json")

	fs["/patches/invalid_version"] = readTestPatch("patch1.json") // Could be any patch

	data, err := ldf.Marshal(updatedBoot)
	if err != nil {
		panic(err)
	}

	fs["/patches/common/a"] = []byte("Test 1")
	fs["/patches/common/b"] = []byte("Test 2")
	fs["/patches/common/c"] = []byte("Test 3")

	fs["/patches/boot.cfg"] = data

	return fs
}

func clientFileSystem() fileSystem {
	fs := make(fileSystem)

	fs["data/file1"] = []byte("default data 1")
	fs["data/file2"] = []byte("default data 2")
	fs["data/file3"] = []byte("default data 3")

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
		return fmt.Errorf("\"%s\" expected `%s` but read `%s`", path, expected, actual)
	}

	return nil
}

func countDirectoryContents(dir string) (int, error) {
	count := 0

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() {
			count++
		}

		return nil
	})

	if err != nil {
		return 0, err
	}

	return count, nil
}

func testPatchVersion(t *testing.T, env *environment, resources client.Resources, version string, clientFS fileSystem, expectedFS fileSystem) {
	t.Log("Initializing client contents:")
	clientFS.Init(env.ClientDir(), t)

	patch, err := env.ServerConfig.GetPatch(version)
	if err != nil {
		t.Fatalf("test patching: %s: %v", version, err)
	}

	err = patch.UpdateResources(env.ServerConfig, env.Rejections)
	if err != nil {
		t.Fatalf("test patching: %s: update resources: %v", version, err)
	}

	err = patch.TransferResources(env.ClientDir(), resources, env.ServerConfig)
	if err != nil {
		t.Fatalf("test patching: %s: transfer resource: %v", version, err)
	}

	numEntries, err := countDirectoryContents(env.ClientDir())
	if err != nil {
		t.Fatalf("test patching: %v", err)
	}

	if len(expectedFS) != numEntries {
		t.Fatalf("test patching: expected %d client entries but got %d", len(expectedFS), numEntries)
	}

	for path, expectedData := range expectedFS {
		err := checkContents(env.ClientDir(), path, expectedData)
		if err != nil {
			t.Errorf("test patching: %v", err)
		} else {
			t.Logf("\"%s\" is correct!", path)
		}
	}
}

func testBadPatchVersion(t *testing.T, env *environment, resources client.Resources, version string, clientFS fileSystem) {
	t.Log("Initializing client contents:")
	clientFS.Init(env.ClientDir(), t)

	patch, err := env.ServerConfig.GetPatch(version)
	if err != nil {
		t.Fatalf("test patching: bad %s: %v", version, err)
	}

	err = patch.TransferResources(env.ClientDir(), resources, env.ServerConfig)
	if err == nil {
		t.Fatalf("test patching: bad %s: transfer resources: did not return an error", version)
	}

	t.Logf("patch %s correctly returned an error! (%v)", version, err)
}

func TestPatching(t *testing.T) {
	expectedBoot := &ldf.BootConfig{
		ServerName:      fmt.Sprintf("Server %d", rand.Uint32()),
		PatchServerIP:   "127.0.0.1",
		PatchServerPort: 3000,
	}

	serverFS := serverFileSystem(expectedBoot)
	clientFS := clientFileSystem()

	env, teardown := setup(t, serverFS)
	defer teardown()

	clientResources := &resources{
		replacements: replacementCache{m: make(map[string]client.Resource)},
		additions:    additionsCache{m: make(map[string]struct{})},
	}

	_, err := env.ServerConfig.GetPatch("v1.0.0")
	if !errors.Is(err, patch.ErrPatchesUnavailable) {
		t.Fatalf("test patching: Server.GetPatch did not return patch.ErrPatchesUnavailable: instead: %v", err)
	}

	go env.PatchServer.ListenAndServe()

	t.Log("Started test patch server.")

	// Test replace directive
	testPatchVersion(t, env, clientResources, "v1.0.0", clientFS, fileSystem{
		"data/file1": []byte("Test 1"),
		"data/file2": []byte("Test 2"),
		"data/file3": []byte("Test 3"),
	})

	// Test add directive
	testPatchVersion(t, env, clientResources, "v2.0.0", clientFS, fileSystem{
		"data/file1": []byte("default data 1"),
		"data/file2": []byte("default data 2"),
		"data/file3": []byte("default data 3"),
		"data/file4": []byte("Test 1"),
		"data/file5": []byte("Test 2"),
		"data/file6": []byte("Test 3"),
	})

	// Test replace and add directive
	testPatchVersion(t, env, clientResources, "v3.0.0", clientFS, fileSystem{
		"data/file1": []byte("Test 1"),
		"data/file2": []byte("Test 2"),
		"data/file3": []byte("default data 3"),
		"data/file4": []byte("Test 3"),
	})

	// Test replacing resources that do not exist
	testBadPatchVersion(t, env, clientResources, "v4.0.0", clientFS)

	// Test adding resources that already exist
	testBadPatchVersion(t, env, clientResources, "v5.0.0", clientFS)

	// Test bad version name
	testBadPatchVersion(t, env, clientResources, "invalid_version", clientFS)

	// Test update directives
	testPatchVersion(t, env, clientResources, "v6.0.0", clientFS, clientFS) // client should remain unchanged

	if env.ServerConfig.Config.ServerName != expectedBoot.ServerName {
		t.Fatalf("test patching: expected ServerName to be \"%s\" but got \"%s\"", env.ServerConfig.Config.ServerName, expectedBoot.ServerName)
	}

	t.Logf("Server name is correct! (\"%s\")", env.ServerConfig.Config.ServerName)

	if env.ServerConfig.PatchProtocol != "proto" {
		t.Fatalf("test patching: expected patchProtocol to be \"updated\" but got \"%s\"", env.ServerConfig.PatchProtocol)
	}

	t.Logf("Patch protocol is correct! (\"%s\")", env.ServerConfig.PatchProtocol)
}
