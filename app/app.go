package app

import (
	"errors"
	"fmt"
	"image/color"
	"log"
	"os"
	"os/exec"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/I-Am-Dench/lu-launcher/resource"
)

type App struct {
	fyne.App
	settings resource.Settings
	servers  resource.ServerList

	main fyne.Window

	serverSelector *widget.Select
	playButton     *widget.Button

	serverNameBinding binding.String
	authServerBinding binding.String
	localeBinding     binding.String

	FoundClient bool
}

func New(settings resource.Settings, servers resource.ServerList) App {
	a := App{}
	a.App = app.New()

	a.settings = settings
	a.servers = servers

	a.main = a.NewWindow("Lego Universe")
	a.main.SetFixedSize(true)
	a.main.Resize(fyne.NewSize(800, 300))

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

	a.serverNameBinding = binding.NewString()
	a.authServerBinding = binding.NewString()
	a.localeBinding = binding.NewString()

	a.LoadContent()

	return a
}

func (app *App) LoadContent() {
	heading := canvas.NewText("Launch Lego Universe", color.White)
	heading.TextSize = 24

	app.serverSelector = widget.NewSelect(
		app.servers.Names(),
		func(s string) {
			app.settings.CurrentServer = app.serverSelector.SelectedIndex()
			err := app.settings.Save()
			if err != nil {
				log.Printf("save settings error: %v\n", err)
			}

			server := app.servers.Get(app.settings.CurrentServer)
			app.serverNameBinding.Set(server.Config.ServerName)
			app.authServerBinding.Set(server.Config.AuthServerIP)
			app.localeBinding.Set(server.Config.Locale)
		},
	)
	app.serverSelector.SetSelectedIndex(app.settings.CurrentServer)

	addServerButton := widget.NewButtonWithIcon(
		"", theme.SettingsIcon(),
		func() {
			log.Println("Heading to settings...")
		},
	)

	serverInfo := widget.NewForm(
		widget.NewFormItem(
			"Server Name", widget.NewLabelWithData(app.serverNameBinding),
		),
		widget.NewFormItem(
			"Auth Server IP", widget.NewLabelWithData(app.authServerBinding),
		),
		widget.NewFormItem(
			"Locale", widget.NewLabelWithData(app.localeBinding),
		),
	)

	innerContent := container.NewPadded(
		container.NewVBox(
			container.NewBorder(
				nil, nil, nil,
				addServerButton,
				app.serverSelector,
			),
			serverInfo,
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
	app.playButton = widget.NewButtonWithIcon(
		"Play",
		theme.MediaPlayIcon(),
		app.PressPlay,
	)

	app.playButton.Importance = widget.HighImportance
	if !app.FoundClient {
		app.playButton.Disable()
	}

	clientLabel := widget.NewLabelWithStyle(
		app.settings.ClientPath(),
		fyne.TextAlignLeading,
		fyne.TextStyle{
			Bold: true,
		},
	)
	clientLabel.Truncation = fyne.TextTruncateEllipsis

	prepareProgressBar := widget.NewProgressBar()
	prepareProgressBar.TextFormatter = func() string {
		return ""
	}

	prepareProgressBar.Hide()

	if app.FoundClient {
		return container.NewBorder(
			prepareProgressBar,
			nil, nil,
			app.playButton,
			clientLabel,
		)
	} else {
		themedIcon := theme.NewErrorThemedResource(theme.ErrorIcon())

		return container.NewBorder(
			prepareProgressBar,
			nil,
			widget.NewIcon(themedIcon),
			app.playButton,
			clientLabel,
		)
	}
}

func (app *App) SetCurrentServer(index int) {
	app.serverSelector.SetSelectedIndex(index)
}

func (app *App) SetPlayingState() {
	app.playButton.Disable()
	app.playButton.SetText("Playing")

	app.serverSelector.Disable()
}

func (app *App) SetNormalState() {
	app.playButton.Enable()
	app.playButton.SetText("Play")
	app.playButton.SetIcon(theme.MediaPlayIcon())
	app.playButton.Importance = widget.HighImportance
	app.playButton.OnTapped = app.PressPlay
	app.playButton.Refresh()

	app.serverSelector.Enable()
}

func (app *App) SetUpdatingState() {
	app.playButton.Disable()
	app.playButton.SetText("Updating")
}

func (app *App) SetUpdateState() {
	app.playButton.Enable()
	app.playButton.SetText("Update")
	app.playButton.SetIcon(theme.DownloadIcon())
	app.playButton.Importance = widget.SuccessImportance
	app.playButton.OnTapped = app.PressUpdate
	app.playButton.Refresh()

}

func (app *App) PressPlay() {
	app.SetPlayingState()
	log.Println("Launching Lego Universe...")

	go func(cmd *exec.Cmd) {
		cmd.Wait()
		app.SetNormalState()
	}(app.StartClient())
}

func (app *App) PressUpdate() {
	log.Println("Starting update...")
	app.SetNormalState()
}

func (app *App) StartClient() *exec.Cmd {
	cmd := exec.Command(app.settings.ClientPath())
	cmd.Dir = app.settings.Client.Directory
	cmd.Start()
	return cmd
}

func (app *App) Start() {
	app.main.CenterOnScreen()
	app.main.ShowAndRun()
}
