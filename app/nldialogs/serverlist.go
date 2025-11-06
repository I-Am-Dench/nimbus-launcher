package nldialogs

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

func ShowServerList(w fyne.Window, servers []string) {

	choices := widget.NewRadioGroup(servers, func(s string) {})
	choices.Required = true

	list := container.NewVScroll(choices)
	list.SetMinSize(list.MinSize().AddWidthHeight(128, 128))

	dialog.ShowCustom("Server List", "Ok", list, w)
}
