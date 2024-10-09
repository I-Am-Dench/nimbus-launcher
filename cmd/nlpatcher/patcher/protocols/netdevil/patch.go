package netdevil

import (
	"fmt"
	"path"
	"path/filepath"

	"github.com/I-Am-Dench/goverbuild/archive/manifest"
	"github.com/I-Am-Dench/goverbuild/models/boot"
	"github.com/I-Am-Dench/nimbus-launcher/cmd/nlpatcher/patcher"
	"github.com/I-Am-Dench/nimbus-launcher/cmd/nlpatcher/patcher/protocols/netdevil/resources"
)

type patchConfig struct {
	patcher.PatchOptions `json:"-"`

	GetResource resources.Func `json:"-"`
	Server      *Server        `json:"-"`

	Locale    string `json:"locale"`
	HighSpeed bool   `json:"highSpeed"`
}

type patch struct {
	patchConfig

	ManifestFile *manifest.Manifest
}

func (patch *patch) Manifest(name string, useVersionsDir ...bool) (*manifest.Manifest, error) {
	if len(useVersionsDir) > 0 && useVersionsDir[0] {
		return manifest.Open(filepath.Join(patch.InstallDirectory, "versions", name))
	} else {
		return manifest.Open(filepath.Join(patch.InstallDirectory, "versions", patch.ServerId, name))
	}
}

func (patch *patch) FetchManifest(name string) (*manifest.Manifest, error) {
	resource, err := patch.GetResource(path.Join(patch.Server.Version, name))
	if err != nil {
		return nil, fmt.Errorf("fetch manifest: %s: %w", name, err)
	}
	defer resource.Close()

	manifest, err := manifest.Read(resource)
	if err != nil {
		return nil, fmt.Errorf("fetch manifest: %s: %w", name, err)
	}

	return manifest, nil
}

func (patch *patch) Patch() (*boot.Config, error) {
	// TODO
	return boot.DefaultConfig, nil
}

func newPatch(config patchConfig) (*patch, error) {
	patch := &patch{
		patchConfig: config,
	}

	manifest, err := patch.FetchManifest("version.txt")
	if err != nil {
		return nil, fmt.Errorf("patch: %w", err)
	}

	patch.ManifestFile = manifest

	return patch, nil
}
