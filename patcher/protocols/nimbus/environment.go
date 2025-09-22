package nimbus

import (
	"context"
	"errors"

	"github.com/I-Am-Dench/nimbus-launcher/patcher"
)

type Environment struct{}

func (e Environment) Locale() string {
	return ""
}

func (e Environment) NewPatcher(ctx context.Context, options patcher.Options) (patcher.Patcher, error) {
	return nil, errors.ErrUnsupported
}
