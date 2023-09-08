package resource

import (
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
)

const (
	assetsDir = "assets"
)

func Of(dir string, name string) string {
	return filepath.Clean(filepath.Join(dir, name))
}

func Asset(name string) (*fyne.StaticResource, error) {
	bytes, err := os.ReadFile(Of(assetsDir, name))
	if err != nil {
		return nil, err
	}

	return fyne.NewStaticResource(name, bytes), nil
}
