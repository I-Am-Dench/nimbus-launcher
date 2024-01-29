package client

import (
	"errors"
	"fmt"
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

type Cache[T any] interface {
	Add(T) error
	Get(key string) (T, error)
	List() ([]T, error)
	Has(key string) bool
}

type Resources interface {
	Replacements() Cache[Resource]
	Additions() Cache[string]

	Close() error
}

func Contains(clientDirectory, resource string) bool {
	_, err := os.Stat(filepath.Join(clientDirectory, resource))
	return !errors.Is(err, os.ErrNotExist)
}

func ReadResource(clientDirectory, resource string) (Resource, error) {
	file, err := os.Open(filepath.Join(clientDirectory, resource))
	if err != nil {
		return Resource{}, fmt.Errorf("client: cannot open resource: %w", err)
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
		return fmt.Errorf("client: cannot write resource: %w", err)
	}

	return os.Chtimes(path, time.Time{}, resource.Time())
}

func RemoveResource(clientDirectory string, path string) error {
	fullPath := filepath.Join(clientDirectory, path)

	stat, err := os.Stat(fullPath)
	if err != nil {
		return fmt.Errorf("client: cannot remove resource: %w", err)
	}

	if stat.IsDir() {
		return fmt.Errorf("client: cannot remove resource: \"%s\" is a directory", fullPath)
	}

	err = os.Remove(fullPath)
	if err != nil {
		return fmt.Errorf("client: cannot remove resource: %w", err)
	}

	return nil
}
