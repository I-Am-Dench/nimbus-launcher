package client

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

type ClientResource struct {
	Path    string
	ModTime int64
	Data    []byte
}

func (resource ClientResource) Time() time.Time {
	return time.Unix(resource.ModTime, 0)
}

type Cache interface {
	Add(resource ClientResource) error
	Get(path string) (ClientResource, error)
	GetResources() ([]ClientResource, error)
	Has(path string) bool
	Close() error
}

func ReadResource(clientDirectory, resource string) (ClientResource, error) {
	file, err := os.Open(filepath.Join(clientDirectory, resource))
	if err != nil {
		return ClientResource{}, fmt.Errorf("clientcache: cannot open resource: %v", err)
	}

	data, _ := io.ReadAll(file)
	stat, _ := file.Stat()
	return ClientResource{
		Path:    filepath.Clean(resource),
		ModTime: stat.ModTime().Unix(),
		Data:    data,
	}, nil
}

func WriteResource(clientDirectory string, resource ClientResource) error {
	path := filepath.Join(clientDirectory, resource.Path)
	err := os.WriteFile(path, resource.Data, 0755)
	if err != nil {
		return fmt.Errorf("clientcache: cannot write resource: %v", err)
	}

	return os.Chtimes(path, time.Time{}, resource.Time())
}
