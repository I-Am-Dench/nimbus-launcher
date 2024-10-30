package archive

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/I-Am-Dench/goverbuild/archive/catalog"
	"github.com/I-Am-Dench/goverbuild/archive/pack"
)

type Archive struct {
	catalog      *catalog.Catalog
	installation string

	packs map[string]*pack.Pack
}

func New(catalog *catalog.Catalog, installation string) *Archive {
	return &Archive{
		catalog:      catalog,
		installation: installation,
		packs:        map[string]*pack.Pack{},
	}
}

func (archive *Archive) FindPack(path string) (*pack.Pack, error) {
	record, ok := archive.catalog.Search(path)
	if !ok {
		return nil, ErrNotCatalogued
	}

	p, ok := archive.packs[strings.ToLower(record.PackName)]
	if ok {
		return p, nil
	}

	p, err := pack.Open(filepath.Join(archive.installation, record.PackName))
	if err != nil {
		return nil, err
	}

	archive.packs[strings.ToLower(record.PackName)] = p
	return p, nil
}

func (archive *Archive) Close() error {
	errs := []error{}
	for _, pack := range archive.packs {
		if err := pack.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
