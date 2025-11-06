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
	InstallDirectory string
	BootConfig       boot.Config
}

func (s Server) Name() string {
	return s.BootConfig.ServerName
}

func (s Server) GetBoot(packed bool) *boot.Config {
	boot := s.BootConfig
	boot.UseCatalog = packed
	return &boot
}

func (s Server) GetVersion(_ context.Context, packed bool) (*archive.Archive, error) {
	if !packed {
		return nil, nil
	}
	return archive.Open(s.InstallDirectory, filepath.Join(s.InstallDirectory, patcher.VersionsDir, patcher.CatalogName))
}

func (s Server) GetPatch(_ context.Context, _ *archive.Archive) (patcher.Patch, error) {
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
