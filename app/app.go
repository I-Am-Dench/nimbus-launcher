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
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/I-Am-Dench/lu-launcher/luconfig"
	"github.com/I-Am-Dench/lu-launcher/resource"
)

type App struct {
	fyne.App
	settings resource.Settings
	servers  resource.ServerList

	main           fyne.Window
	settingsWindow fyne.Window

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
	a.main.SetMaster()

	a.settingsWindow = a.NewWindow("Settings")
	a.settingsWindow.SetFixedSize(true)
	a.settingsWindow.Resize(fyne.NewSize(800, 600))
	a.settingsWindow.Show()
	// a.settingsWindow.Hide()

	icon, err := resource.Asset(IMAGE_ICON)
	if err == nil {
		a.main.SetIcon(icon)
		a.settingsWindow.SetIcon(icon)
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
	a.LoadSettingsContent()

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

func (app *App) LoadSettingsContent() {
	heading := canvas.NewText("Settings", color.White)
	heading.TextSize = 24

	tabs := container.NewAppTabs(
		container.NewTabItem("Servers", app.ServerSettings()),
		container.NewTabItem("Launcher", widget.NewLabel("Launcher settings")),
	)

	app.settingsWindow.SetContent(
		container.NewPadded(
			container.NewVBox(
				heading,
				tabs,
			),
		),
	)
}

func (app *App) ServerSettings() *fyne.Container {
	infoHeading := canvas.NewText("Server Info", color.White)
	infoHeading.TextSize = 16

	serverXML := widget.NewLabel("")
	title := widget.NewEntry()
	patchServer := widget.NewEntry()

	bootHeading := canvas.NewText("boot.cfg", color.White)
	bootHeading.TextSize = 16

	bootForm := NewBootForm()

	serverXMLUpload := widget.NewButtonWithIcon(
		"", theme.FileIcon(),
		func() {
			fileDialog := dialog.NewFileOpen(func(uc fyne.URIReadCloser, err error) {
				if err != nil {
					dialog.ShowError(fmt.Errorf("error when opening server.xml file: %v", err), app.settingsWindow)
					return
				}

				if uc == nil || uc.URI() == nil {
					return
				}
				serverXML.SetText(uc.URI().Path())

				server, err := resource.LoadXML(uc.URI().Path())
				if err != nil {
					dialog.ShowError(err, app.settingsWindow)
					return
				}

				title.SetText(server.Name)
				patchServer.SetText(server.PatchServer)

				bootConfig := luconfig.LUConfig{}
				err = luconfig.Unmarshal([]byte(server.Boot.Text), &bootConfig)
				if err != nil {
					dialog.ShowError(err, app.settingsWindow)
					return
				}

				bootForm.UpdateWith(&bootConfig)
			}, app.settingsWindow)
			fileDialog.SetFilter(storage.NewExtensionFileFilter([]string{".xml"}))
			fileDialog.Show()
		},
	)

	innerContent := container.NewVBox(
		infoHeading,
		widget.NewForm(
			widget.NewFormItem("Server XML", container.NewHBox(serverXMLUpload, serverXML)),
			widget.NewFormItem("Name", title),
			widget.NewFormItem("Patch Server", patchServer),
		),
		widget.NewSeparator(),
		bootHeading,
		bootForm.Container(),
	)

	scrolled := container.NewVScroll(
		innerContent,
	)
	scrolled.SetMinSize(innerContent.MinSize().AddWidthHeight(0, 75))

	addServerButton := widget.NewButton("Add Server", func() {
		config := bootForm.GetConfig()
		server, err := resource.NewServer(title.Text, config)
		if err != nil {
			dialog.ShowError(err, app.settingsWindow)
			return
		}

		err = app.AddServer(server)
		if err != nil {
			dialog.ShowError(err, app.settingsWindow)
			return
		}

		dialog.ShowInformation("Server Added", fmt.Sprintf("Added '%s' to server list!", server.Name), app.settingsWindow)
	})
	addServerButton.Importance = widget.HighImportance

	return container.NewPadded(
		container.NewBorder(
			nil,
			container.NewBorder(nil, nil, nil, addServerButton),
			nil, nil,
			scrolled,
		),
	)
}

func (app *App) SetCurrentServer(index int) {
	app.serverSelector.SetSelectedIndex(index)
}

func (app *App) AddServer(server *resource.Server) error {
	err := app.servers.Add(server)
	if err != nil {
		return err
	}

	app.serverSelector.SetOptions(app.servers.Names())
	return nil
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

func (app *App) ShowSettings() {

}

func (app *App) Start() {
	app.main.CenterOnScreen()
	app.settingsWindow.CenterOnScreen()

	app.main.ShowAndRun()
}
