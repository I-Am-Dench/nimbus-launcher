package patcher

import (
	"context"

	"github.com/I-Am-Dench/goverbuild/archive"
	"github.com/I-Am-Dench/goverbuild/models/boot"
	"github.com/I-Am-Dench/nimbus-launcher/patcher/origin"
	"github.com/I-Am-Dench/nimbus-launcher/patcher/undoer"
)

const (
	VersionsDir = "versions"
	CatalogName = "primary.pki"
)

type Summary struct {
	Header []string
	Rows   [][]string
}

type Patch interface {
	Summary() Summary
	Total() int
	SetProgress(func(n int))
	Run(context.Context, undoer.Undoer) error
}

type Server interface {
	Name() string

	GetBoot(packed bool) *boot.Config
	GetVersion(ctx context.Context, packed bool) (*archive.Archive, error)
	GetPatch(context.Context, *archive.Archive) (Patch, error)
}

type Logger interface {
	Print(v ...any)
	Printf(format string, v ...any)
	Println(v ...any)
}

type Options struct {
	Resources origin.Resources
	Log       Logger

	Index            MasterIndex
	InstallDirectory string
	ServerId         string
}

type Environment interface {
	Locale() string
	GetMasterIndex(ctx context.Context, serviceUrl string, resources origin.Resources) (MasterIndex, error)
	GetServers(ctx context.Context, options Options) ([]Server, error)
}
