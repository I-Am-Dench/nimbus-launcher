package netdevil

import (
	"context"
	"errors"
	"fmt"

	"github.com/I-Am-Dench/goverbuild/archive"
	"github.com/I-Am-Dench/goverbuild/archive/manifest"
	"github.com/I-Am-Dench/nimbus-launcher/patcher"
	"github.com/I-Am-Dench/nimbus-launcher/patcher/undoer"
)

type Patch struct {
	Downloader

	catalog *archive.Catalog
	archive *archive.Archive

	entries []*manifest.Entry
}

func (p *Patch) Archive() *archive.Archive {
	return p.archive
}

func (p *Patch) Summary() []patcher.PatchEntry {
	summary := []patcher.PatchEntry{}

	for _, entry := range p.entries {
		summary = append(summary, patcher.PatchEntry{
			Source: entry.Path,
		})
	}

	return summary
}

func (p *Patch) runPacked(ctx context.Context, undoer undoer.Undoer) error {
	errs := []error{}
	for _, entry := range p.entries {
		if cancelled(ctx) {
			return ctx.Err()
		}

		if err := undoer.Track(entry.Path, p.archive); err != nil {
			errs = append(errs, err)
		}

		if err := p.DownloadPacked(ctx, entry.Path, entry, p.archive); err != nil {
			errs = append(errs, err)
		}
	}

	if err := errors.Join(errs...); err != nil {
		return fmt.Errorf("patch: %v", err)
	}

	return nil
}

func (p *Patch) runUnpacked(ctx context.Context, undoer undoer.Undoer) error {
	errs := []error{}
	for _, entry := range p.entries {
		if cancelled(ctx) {
			return ctx.Err()
		}

		if err := undoer.Track(entry.Path, nil); err != nil {
			errs = append(errs, err)
		}

		reader, err := p.DownloadUnpacked(ctx, entry.Path, entry)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		if err := reader.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if err := errors.Join(errs...); err != nil {
		return fmt.Errorf("patch: %v", err)
	}

	return nil
}

func (p *Patch) Run(ctx context.Context, undoer undoer.Undoer) error {
	if cancelled(ctx) {
		return ctx.Err()
	}

	if p.catalog != nil {
		return p.runPacked(ctx, undoer)
	} else {
		return p.runUnpacked(ctx, undoer)
	}
}

func (p *Patch) Close() error {
	if p.archive != nil {
		if err := p.archive.Close(); err != nil {
			return err
		}
		return p.catalog.Close()
	}
	return nil
}
