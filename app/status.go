package app

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/I-Am-Dench/nimbus-launcher/patcher"
)

type StatusBinding = binding.Item[*patcher.Status]

func NewStatusLabel(data StatusBinding) *fyne.Container {
	onlineIcon := widget.NewIcon(theme.NewSuccessThemedResource(theme.ConfirmIcon()))
	offlineIcon := widget.NewIcon(theme.NewErrorThemedResource(theme.CancelIcon()))

	onlineIcon.Hide()
	offlineIcon.Hide()

	label := widget.NewLabel("")

	data.AddListener(binding.NewDataListener(func() {
		status, _ := data.Get()

		if status == nil {
			onlineIcon.Hide()
			offlineIcon.Hide()
			label.SetText("No Status")
			return
		}

		if !status.Online {
			onlineIcon.Hide()
			offlineIcon.Show()
			label.SetText("Offline")
			return
		}

		onlineIcon.Show()
		offlineIcon.Hide()
		label.SetText("Online")
	}))

	return container.NewHBox(
		container.NewStack(onlineIcon, offlineIcon),
		label,
	)
}
