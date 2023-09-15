package clientcache

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

type ClientCache interface {
	Add(resource ClientResource) error
	Get(path string) (ClientResource, error)
	GetResources() ([]ClientResource, error)
}

func OpenResource(clientDirectory, resource string) (ClientResource, error) {
	file, err := os.Open(filepath.Join(clientDirectory, resource))
	if err != nil {
		return ClientResource{}, fmt.Errorf("clientcache: cannot open resource: %v", err)
	}

	data, _ := io.ReadAll(file)
	stat, _ := file.Stat()
	return ClientResource{
		Path:    resource,
		ModTime: stat.ModTime().Unix(),
		Data:    data,
	}, nil
}
