package nimbus

import (
	"context"
	"errors"

	"github.com/I-Am-Dench/nimbus-launcher/patcher"
	"github.com/I-Am-Dench/nimbus-launcher/patcher/origin"
)

type Environment struct{}

func (e Environment) Locale() string {
	return ""
}

func (e *Environment) GetMasterIndex(ctx context.Context, serviceUrl string, resources origin.Resources) (patcher.MasterIndex, error) {
	return patcher.MasterIndex{}, nil
}

func (e Environment) NewPatcher(ctx context.Context, options patcher.Options) (patcher.Patcher, error) {
	return nil, errors.ErrUnsupported
}
