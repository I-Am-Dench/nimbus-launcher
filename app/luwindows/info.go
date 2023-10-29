package luwindows

import (
	"net/url"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func mustParse(rawUrl string) *url.URL {
	url, err := url.Parse(rawUrl)
	if err != nil {
		panic(err)
	}
	return url
}

func NewInfoWindow(app fyne.App) fyne.Window {
	window := app.NewWindow("Info")
	window.SetIcon(theme.InfoIcon())
	window.SetFixedSize(true)

	heading := canvas.NewText("Info", theme.ForegroundColor())
	heading.TextSize = 16

	window.SetContent(
		container.NewPadded(
			container.NewVBox(
				heading, widget.NewSeparator(),
				widget.NewForm(
					widget.NewFormItem("Author", widget.NewLabel("Theodore Friedrich")),
					widget.NewFormItem("Source", widget.NewHyperlink("https://github.com/I-Am-Dench/lu-launcher", mustParse("https://github.com/I-Am-Dench/lu-launcher"))),
				),
			),
		),
	)

	return window
}
