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

	archive  *archive.Archive
	progress func(int)

	entries []*manifest.Entry
}

func (p *Patch) Summary() patcher.Summary {
	summary := patcher.Summary{
		Header: []string{"Source"},
	}

	for _, entry := range p.entries {
		summary.Rows = append(summary.Rows, []string{entry.Path})
	}

	return summary
}

func (p *Patch) Total() int {
	return len(p.entries)
}

func (p *Patch) SetProgress(progress func(n int)) {
	p.progress = progress
}

func (p *Patch) runPacked(ctx context.Context, undoer undoer.Undoer, archive *archive.Archive) error {
	errs := []error{}
	for i, entry := range p.entries {
		if cancelled(ctx) {
			return ctx.Err()
		}

		if p.progress != nil {
			p.progress(i + 1)
		}

		if err := undoer.Track(entry.Path, archive); err != nil {
			errs = append(errs, err)
		}

		if err := p.DownloadPacked(ctx, entry.Path, entry, archive); err != nil {
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

	if p.archive != nil {
		return p.runPacked(ctx, undoer, p.archive)
	} else {
		return p.runUnpacked(ctx, undoer)
	}
}
