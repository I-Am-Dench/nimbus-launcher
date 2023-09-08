package app

import (
	"errors"
	"fmt"
	"image/color"
	"log"
	"os"
	"path/filepath"

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

	clientDirectory string
	serverSelector  *widget.Select

	FoundClient bool
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

	a.clientDirectory = filepath.Join(
		resource.DefaultAppDataDirectory(),
		DIR_SOFTWARE,
		DIR_UNIVERSE,
	)
	log.Printf("Using \"%s\" as client directory\n", a.clientDirectory)

	_, err = os.Stat(filepath.Join(a.clientDirectory, EXE_CLIENT))
	if a.FoundClient = !errors.Is(err, os.ErrNotExist); a.FoundClient {
		log.Printf("Found valid client \"%s\"\n", EXE_CLIENT)
	} else {
		log.Printf("Cannot find valid executable \"%s\" in client directory: %v", EXE_CLIENT, err)
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
	if !app.FoundClient {
		playButton.Disable()
	}

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
