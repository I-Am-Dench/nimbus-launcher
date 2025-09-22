package patcher

import (
	"context"

	"github.com/I-Am-Dench/goverbuild/models/boot"
)

type PatchEntry struct {
	Source, Destination string
}

type Patch interface {
	Summary() []PatchEntry
}

type Patcher interface {
	GetBoot(packed bool) *boot.Config
	GetPatch(ctx context.Context, packed bool) (Patch, error)
}

type Logger interface {
	Print(v ...any)
	Printf(format string, v ...any)
	Println(v ...any)
}

type Options struct {
	Log Logger

	InstallDirectory string
	ServerId         string
}

type Environment interface {
	Locale() string
	NewPatcher(ctx context.Context, options Options) (Patcher, error)
}
