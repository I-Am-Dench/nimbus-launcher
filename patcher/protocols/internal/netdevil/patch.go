package netdevil

import (
	"context"
	"errors"

	"github.com/I-Am-Dench/goverbuild/archive"
	"github.com/I-Am-Dench/goverbuild/archive/manifest"
	"github.com/I-Am-Dench/nimbus-launcher/patcher"
	"github.com/I-Am-Dench/nimbus-launcher/patcher/tracker"
)

type Patch struct {
	Downloader

	ar       *archive.Archive
	progress func(int)

	entries []manifest.Entry
}

func (p Patch) Summary() patcher.Summary {
	summary := patcher.Summary{
		Header: []string{"Source"},
	}

	for _, entry := range p.entries {
		summary.Rows = append(summary.Rows, []string{entry.Path})
	}

	return summary
}

func (p Patch) Total() int {
	return len(p.entries)
}

func (p *Patch) SetProgress(progress func(n int)) {
	p.progress = progress
}

func (p Patch) runPacked(ctx context.Context, tracker tracker.Tracker, ar *archive.Archive) error {
	errs := []error{}
	for i, entry := range p.entries {
		if Cancelled(ctx) {
			return ctx.Err()
		}

		if p.progress != nil {
			p.progress(i + 1)
		}

		if err := tracker.Track(entry.Path, ar); err != nil {
			errs = append(errs, err)
		}

		if err := p.DownloadCataloged(ctx, entry.Path, entry, ar); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

func (p Patch) runUnpacked(ctx context.Context, tracker tracker.Tracker) error {
	errs := []error{}
	for i, entry := range p.entries {
		if Cancelled(ctx) {
			return ctx.Err()
		}

		if p.progress != nil {
			p.progress(i + 1)
		}

		if err := tracker.Track(entry.Path, nil); err != nil {
			errs = append(errs, err)
		}

		reader, err := p.DownloadUncataloged(ctx, entry.Path, entry)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		if err := reader.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

func (p Patch) Run(ctx context.Context, tkr tracker.Tracker) error {
	if Cancelled(ctx) {
		return ctx.Err()
	}

	if tkr == nil {
		tkr = tracker.Discard{}
	}

	if p.ar != nil {
		return p.runPacked(ctx, tkr, p.ar)
	} else {
		return p.runUnpacked(ctx, tkr)
	}
}
