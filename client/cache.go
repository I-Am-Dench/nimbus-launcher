package client

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"time"
)

type Resource struct {
	Path    string
	ModTime int64
	Data    []byte
}

func (resource Resource) Time() time.Time {
	return time.Unix(resource.ModTime, 0)
}

type Cache interface {
	Add(resource Resource) error
	Get(path string) (Resource, error)
	GetResources() ([]Resource, error)
	Has(path string) bool
	Close() error
}

func Contains(clientDirectory, resource string) bool {
	_, err := os.Stat(filepath.Join(clientDirectory, resource))
	return !errors.Is(err, os.ErrNotExist)
}

func ReadResource(clientDirectory, resource string) (Resource, error) {
	file, err := os.Open(filepath.Join(clientDirectory, resource))
	if err != nil {
		return Resource{}, &ResourceError{"client: cannot open resource", err}
	}
	defer file.Close()

	data, _ := io.ReadAll(file)
	stat, _ := file.Stat()
	return Resource{
		Path:    filepath.Clean(resource),
		ModTime: stat.ModTime().Unix(),
		Data:    data,
	}, nil
}

func WriteResource(clientDirectory string, resource Resource) error {
	path := filepath.Join(clientDirectory, resource.Path)
	err := os.WriteFile(path, resource.Data, 0755)
	if err != nil {
		return &ResourceError{"client: cannot write resource", err}
	}

	return os.Chtimes(path, time.Time{}, resource.Time())
}
