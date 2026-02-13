package app

import (
	"fmt"
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/I-Am-Dench/nimbus-launcher/patcher"
)

type StatusBinding = binding.Item[*patcher.Status]

func showStatus(window fyne.Window, data StatusBinding) func() {
	return func() {
		status, _ := data.Get()
		items := []*widget.FormItem{}

		if len(status.Version) > 0 {
			items = append(items, widget.NewFormItem("Version", widget.NewLabel(status.Version)))
		}

		if status.DataCenterId > 0 {
			items = append(items, widget.NewFormItem("Data Center ID", widget.NewLabel(strconv.Itoa(status.DataCenterId))))
		}

		if !status.LastUpdate.IsZero() {
			items = append(items, widget.NewFormItem("Last Update", widget.NewLabel(status.LastUpdate.Format(time.RFC822))))
		}

		if status.Online {
			if uptime := status.FormatUptimeSince(time.Now()); len(uptime) > 0 {
				items = append(items, widget.NewFormItem("Uptime", widget.NewLabel(uptime)))
			}
		}

		form := container.NewHScroll(widget.NewForm(items...))
		form.SetMinSize(fyne.NewSize(320, 64))

		dialog.ShowCustom("Server Status", "Ok", form, window)
	}
}

func NewStatusWidget(window fyne.Window, data StatusBinding) *fyne.Container {
	onlineButton := widget.NewButtonWithIcon("Online", theme.NewSuccessThemedResource(theme.ConfirmIcon()), showStatus(window, data))
	offlineButton := widget.NewButtonWithIcon("Offline", theme.NewErrorThemedResource(theme.CancelIcon()), showStatus(window, data))
	noStatus := widget.NewLabel("No Status")

	onlinePlayers := widget.NewLabel("")
	onlinePlayers.Alignment = fyne.TextAlignTrailing

	onlineButton.Importance = widget.LowImportance
	offlineButton.Importance = widget.LowImportance

	onlineButton.Hide()
	offlineButton.Hide()

	data.AddListener(binding.NewDataListener(func() {
		status, _ := data.Get()

		if status == nil {
			onlinePlayers.Hide()
			onlineButton.Hide()
			offlineButton.Hide()
			noStatus.Show()
			return
		}

		if !status.Online {
			onlinePlayers.Hide()
			onlineButton.Hide()
			offlineButton.Show()
			noStatus.Hide()
		} else {
			onlinePlayersText := ""
			if status.OnlineUsersMax == 0 {
				onlinePlayersText = fmt.Sprint(status.OnlineUsers, " Players")
			} else {
				onlinePlayersText = fmt.Sprint(status.OnlineUsers, "/", status.OnlineUsersMax, " Players")
			}
			onlinePlayers.SetText(onlinePlayersText)
			onlinePlayers.Show()

			onlineButton.Show()
			offlineButton.Hide()
			noStatus.Hide()
		}
	}))

	return container.NewHBox(
		container.NewStack(onlineButton, offlineButton, noStatus),
		onlinePlayers,
	)
}
