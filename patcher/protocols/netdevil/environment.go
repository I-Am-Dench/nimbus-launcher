package netdevil

import (
	"context"
	"errors"

	"github.com/I-Am-Dench/nimbus-launcher/patcher"
)

type UserConfig struct {
	Locale       string `json:"locale" xml:"locale"`
	FullDownload bool   `json:"fullDownload" xml:"-"`
}

type Environment struct {
	Environment string `json:"environment" xml:"environment"`
	UserConfig
}

func (e Environment) Locale() string {
	return e.UserConfig.Locale
}

func (e Environment) NewPatcher(ctx context.Context, options patcher.Options) (patcher.Patcher, error) {
	return nil, errors.New("not implemented")
}
