package netdevil

import (
	"context"
	"errors"
	"fmt"

	"github.com/I-Am-Dench/goverbuild/archive/manifest"
	"github.com/I-Am-Dench/nimbus-launcher/patcher"
)

type Patch struct {
	entries      []*manifest.Entry
	downloadFunc func(path string, entry *manifest.Entry) error
}

func (patch *Patch) cancelled(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}

func (patch *Patch) Run(ctx context.Context) error {
	if patch.cancelled(ctx) {
		return fmt.Errorf("patch: %w", ctx.Err())
	}

	if patch.downloadFunc == nil {
		return nil
	}

	errs := []error{}
	for _, entry := range patch.entries {
		if patch.cancelled(ctx) {
			return fmt.Errorf("patch: %w", ctx.Err())
		}

		if err := patch.downloadFunc(entry.Path, entry); err != nil {
			errs = append(errs, err)
		}
	}

	if err := errors.Join(errs...); err != nil {
		return fmt.Errorf("patch: %w", err)
	}

	return nil
}

func (patch *Patch) Summary() []patcher.PatchEntry {
	summary := []patcher.PatchEntry{}

	for _, entry := range patch.entries {
		summary = append(summary, patcher.PatchEntry{
			Source: entry.Path,
		})
	}

	return summary
}
