package patcher

import (
	"io"

	"github.com/I-Am-Dench/goverbuild/models/boot"
)

type Patch interface {
	Patch() (*boot.Config, error)
}

type Patcher interface {
	Authenticate() (bool, error)
	GetPatch(PatchOptions) (Patch, error)
}

type Func = func(io.Reader, Config) (Patcher, error)
