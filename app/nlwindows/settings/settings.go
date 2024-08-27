package settings

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"github.com/I-Am-Dench/nimbus-launcher/app/multiwindow"
	"github.com/I-Am-Dench/nimbus-launcher/app/nlwidgets"
	"github.com/I-Am-Dench/nimbus-launcher/resource"
)

func New(app *multiwindow.App, id int, list *nlwidgets.ServerList, settings *resource.Settings, onSave func()) fyne.Window {
	window := app.NewInstanceWindow("Settings", id)
	window.SetFixedSize(true)
	window.Resize(fyne.NewSize(800, 600))
	window.SetIcon(theme.StorageIcon())

	heading := canvas.NewText("Settings", theme.ForegroundColor())
	heading.TextSize = 24

	window.SetContent(
		container.NewPadded(
			container.NewBorder(
				heading, nil, nil, nil,
				container.NewAppTabs(
					NewServersTab(window, list),
					NewLauncherTab(window, settings, onSave),
				),
			),
		),
	)

	return window
}
