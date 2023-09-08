package app

import (
	"fmt"
	"image/color"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/I-Am-Dench/lu-launcher/resource"
)

type App struct {
	fyne.App

	main fyne.Window

	serverSelector *widget.Select
}

func New() App {
	a := App{}
	a.App = app.New()

	a.main = a.NewWindow("Lego Universe")
	a.main.SetFixedSize(true)
	a.main.Resize(fyne.NewSize(800, 600))

	icon, err := resource.Asset(IMAGE_ICON)
	if err == nil {
		a.main.SetIcon(icon)
	} else {
		log.Println(fmt.Errorf("unable to load icon: %v", err))
	}

	a.LoadContent()

	return a
}

func (app *App) LoadContent() {
	heading := canvas.NewText("Launch Lego Universe", color.White)
	heading.TextSize = 24

	app.serverSelector = widget.NewSelect(
		[]string{"Local", "Theo's Crib"},
		func(s string) {},
	)
	app.serverSelector.SetSelectedIndex(0)

	playButton := widget.NewButtonWithIcon(
		"Play",
		theme.MediaPlayIcon(),
		func() {
			log.Printf("Launching %s\n", app.serverSelector.Selected)
		},
	)
	playButton.Importance = widget.HighImportance

	innerContent := container.NewVBox(
		container.NewPadded(
			app.serverSelector,
		),
	)

	app.main.SetContent(
		container.NewPadded(
			container.NewBorder(
				heading,
				playButton,
				nil, nil,
				innerContent,
			),
		),
	)
}

func (app *App) SetCurrentServer(index int) {
	app.serverSelector.SetSelectedIndex(index)
}

func (app *App) Start() {
	app.main.CenterOnScreen()
	app.main.ShowAndRun()
}
