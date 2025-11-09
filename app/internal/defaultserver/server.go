package defaultserver

import (
	"context"
	"path/filepath"

	"github.com/I-Am-Dench/goverbuild/archive"
	"github.com/I-Am-Dench/goverbuild/models/boot"
	"github.com/I-Am-Dench/nimbus-launcher/patcher"
	"github.com/I-Am-Dench/nimbus-launcher/patcher/undoer"
)

var _ patcher.Server = (*Server)(nil)

type Server struct {
	BootConfig boot.Config
}

func (s Server) Info() patcher.ServerInfo {
	return patcher.ServerInfo{
		Name:   s.BootConfig.ServerName,
		Lang:   s.BootConfig.Locale,
		AuthIP: s.BootConfig.AuthServerIP,
	}
}

func (s Server) GetPatcher(o patcher.Options) patcher.Patcher {
	return Patcher{
		InstallDirectory: o.InstallDirectory,
		BootConfig:       s.BootConfig,
	}
}

type Patcher struct {
	InstallDirectory string
	BootConfig       boot.Config
}

func (p Patcher) GetBoot(packed bool) boot.Config {
	config := p.BootConfig
	config.UseCatalog = packed
	return config
}

func (p Patcher) GetVersion(_ context.Context, packed bool) (*archive.Archive, error) {
	if !packed {
		return nil, nil
	}
	return archive.Open(p.InstallDirectory, filepath.Join(p.InstallDirectory, patcher.VersionsDir, patcher.CatalogName))
}

func (p Patcher) GetPatch(_ context.Context, _ *archive.Archive) (patcher.Patch, error) {
	return Patch{}, nil
}

type Patch struct{}

func (p Patch) Summary() patcher.Summary {
	return patcher.Summary{
		Header: []string{},
		Rows:   [][]string{},
	}
}

func (p Patch) Total() int {
	return 0
}

func (p Patch) SetProgress(_ func(n int)) {}

func (p Patch) Run(_ context.Context, _ undoer.Undoer) error {
	return nil
}
