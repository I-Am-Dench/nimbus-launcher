//go:build windows
// +build windows

package app

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"github.com/I-Am-Dench/nimbus-launcher/client"
)

func DefaultSettings() *Settings {
	return &Settings{
		Launch: LaunchConfig{
			DefaultClient:             client.DefaultConfig,
			CloseOnPlay:               true,
			ReviewPatchesBeforeUpdate: true,
		},
	}
}

func NewEtcSettings(windows fyne.Window, settings *Settings) (*fyne.Container, func() struct{}) {
	return container.NewStack(), func() struct{} { return struct{}{} }
}
