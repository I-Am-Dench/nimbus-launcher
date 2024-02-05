package nlwindows

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
)

func NewSettingsWindow(app fyne.App, tabsFunc func(fyne.Window) []*container.TabItem) fyne.Window {
	window := app.NewWindow("Settings")
	window.SetFixedSize(true)
	window.Resize(fyne.NewSize(800, 600))
	window.SetIcon(theme.StorageIcon())

	heading := canvas.NewText("Settings", theme.ForegroundColor())
	heading.TextSize = 24

	window.SetContent(
		container.NewPadded(
			container.NewBorder(
				heading, nil, nil, nil,
				container.NewAppTabs(tabsFunc(window)...),
			),
		),
	)

	return window
}
