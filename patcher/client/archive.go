package client

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/I-Am-Dench/goverbuild/archive/catalog"
	"github.com/I-Am-Dench/goverbuild/archive/pack"
)

var (
	ErrNotCataloged  = errors.New("not cataloged")
	ErrPackNotExists = errors.New("pack does not exist")
)

type Archive struct {
	Catalog      *catalog.Catalog
	Installation string

	packs map[string]*pack.Pack
}

func NewArchive(catalog *catalog.Catalog, installation string) *Archive {
	return &Archive{
		Catalog:      catalog,
		Installation: installation,
		packs:        make(map[string]*pack.Pack),
	}
}

func (a *Archive) FindPack(path string) (*pack.Pack, error) {
	record, ok := a.Catalog.Search(path)
	if !ok {
		return nil, ErrNotCataloged
	}

	p, ok := a.packs[strings.ToLower(record.PackName)]
	if ok {
		return p, nil
	}

	p, err := pack.Open(filepath.Join(a.Installation, record.PackName))
	if errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("open pack: %s: %w", record.PackName, ErrPackNotExists)
	}

	if err != nil {
		return nil, err
	}

	a.packs[strings.ToLower(record.PackName)] = p
	return p, nil
}

func (a *Archive) Close() error {
	errs := []error{}
	for _, pack := range a.packs {
		if err := pack.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
