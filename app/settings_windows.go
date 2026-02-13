//go:build windows
// +build windows

package app

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"github.com/I-Am-Dench/nimbus-launcher/client"
)

func DefaultSettings() Settings {
	return Settings{
		Launch: LaunchConfig{
			DefaultClient:             client.DefaultConfig,
			CloseOnPlay:               true,
			ReviewPatchesBeforeUpdate: true,
		},
	}
}

func NewEtcSettings(_ fyne.Window, _ Settings) (*fyne.Container, func() client.Etc) {
	return container.NewStack(), func() client.Etc { return client.Etc{} }
}
