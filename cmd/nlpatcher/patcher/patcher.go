package patcher

import (
	"io"
	"path/filepath"
	"strings"

	"github.com/I-Am-Dench/goverbuild/models/boot"
)

type Patch interface {
	Patch() (*boot.Config, error)
}

type Patcher interface {
	Authenticate() (bool, error)
	GetPatch(PatchOptions) (Patch, bool, error)
}

type Func = func(io.Reader, Config) (Patcher, error)

func FileSchemeToPath(uri string) string {
	return filepath.FromSlash(strings.TrimPrefix(uri, "/"))
}
