package app

import (
	"errors"
	"fmt"
	"image/color"
	"log"
	"os"

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
	settings resource.Settings

	main fyne.Window

	// clientDirectory string
	serverSelector *widget.Select

	FoundClient bool
}

func New(settings resource.Settings) App {
	a := App{}
	a.App = app.New()

	a.settings = settings

	a.main = a.NewWindow("Lego Universe")
	a.main.SetFixedSize(true)
	a.main.Resize(fyne.NewSize(800, 600))

	icon, err := resource.Asset(IMAGE_ICON)
	if err == nil {
		a.main.SetIcon(icon)
	} else {
		log.Println(fmt.Errorf("unable to load icon: %v", err))
	}

	log.Printf("Using \"%s\" as client directory\n", a.settings.Client.Directory)

	_, err = os.Stat(a.settings.ClientPath())
	if a.FoundClient = !errors.Is(err, os.ErrNotExist); a.FoundClient {
		log.Printf("Found valid client \"%s\"\n", a.settings.Client.Name)
	} else {
		log.Printf("Cannot find valid executable \"%s\" in client directory: %v", a.settings.Client.Name, err)
	}

	a.LoadContent()

	return a
}

func (app *App) LoadContent() {
	heading := canvas.NewText("Launch Lego Universe", color.White)
	heading.TextSize = 24

	app.serverSelector = widget.NewSelect(
		[]string{"Local", "Theo's Crib"},
		func(s string) {
			app.settings.CurrentServer = app.serverSelector.SelectedIndex()
			err := app.settings.Save()
			if err != nil {
				log.Printf("save settings error: %v\n", err)
			}
		},
	)
	app.serverSelector.SetSelectedIndex(app.settings.CurrentServer)

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
				app.Footer(),
				nil, nil,
				innerContent,
			),
		),
	)
}

func (app *App) Footer() *fyne.Container {
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

	clientLabel := widget.NewLabelWithStyle(
		app.settings.ClientPath(),
		fyne.TextAlignLeading,
		fyne.TextStyle{
			Bold: true,
		},
	)
	clientLabel.Truncation = fyne.TextTruncateEllipsis

	if app.FoundClient {
		return container.NewBorder(
			nil, nil, nil,
			playButton,
			clientLabel,
		)
	} else {
		themedIcon := theme.NewErrorThemedResource(theme.ErrorIcon())

		return container.NewBorder(
			nil, nil,
			widget.NewIcon(themedIcon),
			playButton,
			clientLabel,
		)
	}
}

func (app *App) SetCurrentServer(index int) {
	app.serverSelector.SetSelectedIndex(index)
}

func (app *App) Start() {
	app.main.CenterOnScreen()
	app.main.ShowAndRun()
}
