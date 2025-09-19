//go:build linux
// +build linux

package app

import (
	"log/slog"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/I-Am-Dench/nimbus-launcher/app/nlwidgets"
	"github.com/I-Am-Dench/nimbus-launcher/client"
)

func DefaultSettings() *Settings {
	home, err := os.UserHomeDir()
	if err != nil {
		slog.Error(err.Error())
	}

	clientConfig := client.DefaultConfig
	clientConfig.Etc = client.Steam{
		SteamHome:  filepath.Join(home, ".steam", "steam"),
		CompatData: "{InstallationDir}/.proton",
	}

	return &Settings{
		Launch: LaunchConfig{
			DefaultClient:             clientConfig,
			CloseOnPlay:               true,
			ReviewPatchesBeforeUpdate: true,
		},
	}
}

func NewEtcSettings(window fyne.Window, settings *Settings) (*fyne.Container, func() client.Etc) {
	protonSelector := widget.NewSelect(settings.Launch.DefaultClient.Etc.ProtonOptions(), func(s string) {})
	protonSelector.PlaceHolder = "(Select Proton)"
	protonSelector.SetSelected(settings.Launch.DefaultClient.Etc.Proton)

	updateOptions := func(path string) {
		s := client.Steam{SteamHome: path}
		protonSelector.ClearSelected()
		protonSelector.SetOptions(s.ProtonOptions())
	}

	steamHome := nlwidgets.NewDirectorySelector(window, updateOptions)
	steamHome.PlaceHolder = ".steam/steam"
	steamHome.OnSubmitted = updateOptions
	steamHome.SetText(settings.Launch.DefaultClient.Etc.SteamHome)

	compatData := nlwidgets.NewDirectorySelector(window)
	compatData.PlaceHolder = ".proton"
	compatData.SetText(settings.Launch.DefaultClient.Etc.CompatData)

	header := canvas.NewText("Proton", theme.Color(theme.ColorNameForeground))
	header.TextSize = 16

	return container.NewVBox(
			header,
			widget.NewForm(
				widget.NewFormItem("Proton", protonSelector),
				widget.NewFormItem("Steam Home", steamHome),
				widget.NewFormItem("Compat Data", compatData),
			),
		), func() client.Etc {
			return client.Steam{
				Proton:     protonSelector.Selected,
				SteamHome:  steamHome.Text,
				CompatData: compatData.Text,
			}
		}
}
