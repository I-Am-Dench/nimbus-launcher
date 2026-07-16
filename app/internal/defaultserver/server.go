package defaultserver

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/I-Am-Dench/goverbuild/archive"
	"github.com/I-Am-Dench/goverbuild/encoding/ldf"
	"github.com/I-Am-Dench/goverbuild/models/boot"
	"github.com/I-Am-Dench/nimbus-launcher/patcher"
	"github.com/I-Am-Dench/nimbus-launcher/patcher/tracker"
)

var _ patcher.Server = (*Server)(nil)

type BootConfig struct {
	boot.Config
	ldf.Map
}

type Server struct {
	BootConfig BootConfig
}

func (s Server) Info() patcher.ServerInfo {
	return patcher.ServerInfo{
		Name:   s.BootConfig.ServerName,
		Lang:   s.BootConfig.Locale,
		AuthIP: s.BootConfig.AuthServerIP,
	}
}

func (s Server) Status() *patcher.Status {
	return nil
}

func (s Server) GetPatcher(o patcher.Options) (patcher.Patcher, error) {
	return Patcher{
		InstallDirectory: o.InstallDirectory,
		BootConfig:       s.BootConfig,
	}, nil
}

type Patcher struct {
	InstallDirectory string
	BootConfig       BootConfig
}

func (p Patcher) GetBoot(packed bool) (boot.Config, ldf.Map) {
	config := p.BootConfig
	config.UseCatalog = packed
	return config.Config, p.BootConfig.Map
}

func (p Patcher) GetVersion(_ context.Context, packed bool) (*archive.Archive, error) {
	if !packed {
		return nil, nil
	}

	catalogName := filepath.Join(p.InstallDirectory, patcher.VersionsDir, patcher.CatalogName)

	ar, err := archive.Open(p.InstallDirectory, catalogName)
	if err != nil {
		return nil, fmt.Errorf("patcher is configured to a packed installation, but failed to open catalog: %v", catalogName)
	}
	return ar, nil
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

func (p Patch) Run(_ context.Context, _ tracker.Tracker) error {
	return nil
}
