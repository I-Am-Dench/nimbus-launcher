package netdevil

import (
	"context"
	"errors"
	"fmt"

	"github.com/I-Am-Dench/goverbuild/archive/catalog"
	"github.com/I-Am-Dench/goverbuild/archive/manifest"
	"github.com/I-Am-Dench/nimbus-launcher/patcher"
	"github.com/I-Am-Dench/nimbus-launcher/patcher/client"
	"github.com/I-Am-Dench/nimbus-launcher/patcher/undoer"
)

type Patch struct {
	Downloader

	catalog *catalog.Catalog
	entries []*manifest.Entry
}

func (p *Patch) Catalog() (*catalog.Catalog, bool) {
	return p.catalog, p.catalog != nil
}

func (p *Patch) runPacked(ctx context.Context, undoer undoer.Undoer) (err error) {
	archive := client.NewArchive(p.catalog, p.Root)

	errs := []error{}
	for _, entry := range p.entries {
		if cancelled(ctx) {
			archive.Close()
			return fmt.Errorf("patch: %w", ctx.Err())
		}

		if err := undoer.Track(entry.Path, archive); err != nil {
			errs = append(errs, err)
		}

		if err := p.DownloadPacked(ctx, entry.Path, entry, archive); err != nil {
			errs = append(errs, err)
		}
	}

	if err := archive.Close(); err != nil {
		errs = append(errs, err)
	}

	if err := errors.Join(errs...); err != nil {
		return fmt.Errorf("patch: %w", err)
	}

	return nil
}

func (p *Patch) runUnpacked(ctx context.Context, undoer undoer.Undoer) error {
	errs := []error{}
	for _, entry := range p.entries {
		if cancelled(ctx) {
			return fmt.Errorf("patch: %w", ctx.Err())
		}

		if err := undoer.Track(entry.Path, nil); err != nil {
			errs = append(errs, err)
		}

		r, err := p.DownloadUnpacked(ctx, entry.Path, entry)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		if err := r.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if err := errors.Join(errs...); err != nil {
		return fmt.Errorf("patch: %w", err)
	}

	return nil
}

func (p *Patch) Run(ctx context.Context, undoer undoer.Undoer) error {
	if cancelled(ctx) {
		return fmt.Errorf("patch: %w", ctx.Err())
	}

	if p.catalog != nil {
		return p.runPacked(ctx, undoer)
	} else {
		return p.runUnpacked(ctx, undoer)
	}
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

// type Patch struct {
// 	Downloader

// 	entries []*manifest.Entry
// 	// downloadFunc func(path string, entry *manifest.Entry, undoer undoer.Undoer) error
// }

// // func (patch *Patch) cancelled(ctx context.Context) bool {
// // 	select {
// // 	case <-ctx.Done():
// // 		return true
// // 	default:
// // 		return false
// // 	}
// // }

// func (patch *Patch) Run(ctx context.Context, undoer undoer.Undoer) error {
// 	if patch.Cancelled() {
// 		return fmt.Errorf("patch: %w", ctx.Err())
// 	}

// 	// if patch.downloadFunc == nil {
// 	// 	return nil
// 	// }

// 	errs := []error{}
// 	for _, entry := range patch.entries {
// 		if patch.Cancelled() {
// 			return fmt.Errorf("patch: %w", ctx.Err())
// 		}

// 		if err := patch.downloadFunc(entry.Path, entry, undoer); err != nil {
// 			errs = append(errs, err)
// 		}
// 	}

// 	if err := errors.Join(errs...); err != nil {
// 		return fmt.Errorf("patch: %w", err)
// 	}

// 	return nil
// }

// func (patch *Patch) Summary() []patcher.PatchEntry {
// 	summary := []patcher.PatchEntry{}

// 	for _, entry := range patch.entries {
// 		summary = append(summary, patcher.PatchEntry{
// 			Source: entry.Path,
// 		})
// 	}

// 	return summary
// }
