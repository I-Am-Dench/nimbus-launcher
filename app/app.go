package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"image/color"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"

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

var (
	ErrPatchesUnsupported = errors.New("patches unsupported")
	ErrPatchesUnavailable = errors.New("patch server could not be reached")
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

	serverPatches map[string]resource.Patches

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

	a.serverPatches = make(map[string]resource.Patches)

	a.LoadContent()

	return a
}

func (app *App) LoadContent() {
	heading := canvas.NewText("Launch Lego Universe", color.White)
	heading.TextSize = 24

	app.serverSelector = widget.NewSelect(
		app.servers.Names(),
		func(s string) {
			app.SetCurrentServer(app.serverSelector.SelectedIndex())
		},
	)
	app.serverSelector.SetSelectedIndex(app.settings.CurrentServer)

	addServerButton := widget.NewButtonWithIcon(
		"", theme.SettingsIcon(), app.ShowSettings,
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

func (app *App) LoadSettingsContent(window fyne.Window) {
	heading := canvas.NewText("Settings", color.White)
	heading.TextSize = 24

	tabs := container.NewAppTabs(
		container.NewTabItem("Servers", app.ServerSettings(window)),
		container.NewTabItem("Launcher", app.LauncherSettings(window)),
	)

	window.SetContent(
		container.NewPadded(
			container.NewBorder(
				heading, nil, nil, nil,
				tabs,
			),
		),
	)
}

func (app *App) ServerSettings(window fyne.Window) *fyne.Container {
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
					dialog.ShowError(fmt.Errorf("error when opening server.xml file: %v", err), window)
					return
				}

				if uc == nil || uc.URI() == nil {
					return
				}
				serverXML.SetText(uc.URI().Path())

				server, err := resource.LoadXML(uc.URI().Path())
				if err != nil {
					dialog.ShowError(err, window)
					return
				}

				title.SetText(server.Name)
				patchServer.SetText(server.PatchServer)

				bootConfig := luconfig.LUConfig{}
				err = luconfig.Unmarshal([]byte(server.Boot.Text), &bootConfig)
				if err != nil {
					dialog.ShowError(err, window)
					return
				}

				bootForm.UpdateWith(&bootConfig)
			}, window)
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

	addServerButton := widget.NewButton("Add Server", func() {
		config := bootForm.GetConfig()
		server, err := resource.NewServer(title.Text, config)
		if err != nil {
			dialog.ShowError(err, window)
			return
		}

		err = app.AddServer(server)
		if err != nil {
			dialog.ShowError(err, window)
			return
		}

		dialog.ShowInformation("Server Added", fmt.Sprintf("Added '%s' to server list!", server.Name), window)
	})
	addServerButton.Importance = widget.HighImportance

	return container.NewPadded(
		container.NewBorder(
			nil,
			container.NewBorder(nil, nil, nil, addServerButton),
			nil, nil,
			container.NewVScroll(
				innerContent,
			),
		),
	)
}

func (app *App) LauncherSettings(window fyne.Window) *fyne.Container {
	generalHeading := canvas.NewText("General", color.White)
	generalHeading.TextSize = 16

	closeOnPlay := widget.NewCheck("", func(b bool) {})
	closeOnPlay.Checked = app.settings.CloseOnPlay

	saveButton := widget.NewButton("Save", func() {
		app.settings.CloseOnPlay = closeOnPlay.Checked

		err := app.settings.Save()
		if err != nil {
			dialog.ShowError(err, window)
		} else {
			dialog.ShowInformation("Launcher Settings", "Settings saved!", window)
		}
	})
	saveButton.Importance = widget.HighImportance

	return container.NewPadded(
		container.NewBorder(
			nil,
			container.NewBorder(nil, nil, nil, saveButton),
			nil, nil,
			container.NewVScroll(
				container.NewVBox(
					generalHeading,
					widget.NewForm(
						widget.NewFormItem("Close Launcher When Played", closeOnPlay),
					),
				),
			),
		),
	)
}

func (app *App) SetCurrentServerInfo(server *resource.Server) {
	app.serverNameBinding.Set(server.Config.ServerName)
	app.authServerBinding.Set(server.Config.AuthServerIP)
	app.localeBinding.Set(server.Config.Locale)
}

func (app *App) SetCurrentServer(index int) {
	app.settings.CurrentServer = index

	err := app.settings.Save()
	if err != nil {
		log.Printf("save setings err: %v\n", err)
	}

	server := app.servers.Get(app.settings.CurrentServer)
	app.SetCurrentServerInfo(server)

	// If it's nil, the app has not started yet
	if app.playButton != nil {
		app.CheckForUpdates(server)
	}
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

func (app *App) SetCheckingUpdatesState() {
	app.playButton.Disable()
	app.playButton.SetText("Checking updates")
	app.playButton.SetIcon(theme.ViewRefreshIcon())
	app.playButton.Refresh()
}

func (app *App) CurrentServer() *resource.Server {
	return app.servers.Get(app.settings.CurrentServer)
}

func (app *App) CopyBootConfiguration(server *resource.Server) error {
	data, err := os.ReadFile(server.BootPath())
	if err != nil {
		return fmt.Errorf("cannot read \"%s\": %v", server.BootPath(), err)
	}

	configPath := filepath.Join(app.settings.Client.Directory, "boot.cfg")
	return os.WriteFile(configPath, data, 0755)
}

func (app *App) PressPlay() {
	app.SetPlayingState()
	log.Printf("Selected server: %s\n", app.CurrentServer().Name)

	if app.settings.CurrentServer != app.settings.PreviouslyRunServer {
		log.Println("Selected server does not match previously run server; Copying over boot.cfg")
		err := app.CopyBootConfiguration(app.CurrentServer())
		if err != nil {
			dialog.ShowError(fmt.Errorf("could not copy \"boot.cfg\": %v", err), app.main)
			app.SetNormalState()
			return
		}
		log.Println("Copy completed.")
	}

	app.settings.PreviouslyRunServer = app.settings.CurrentServer
	app.settings.Save()

	log.Println("Launching Lego Universe...")
	log.Printf("Close launcher when played: %v\n", app.settings.CloseOnPlay)

	cmd := app.StartClient()
	if app.settings.CloseOnPlay {
		app.main.Close()
		return
	}

	go func(cmd *exec.Cmd) {
		cmd.Wait()
		app.SetNormalState()
	}(cmd)
}

func (app *App) PressUpdate() {
	app.Update(app.CurrentServer())
}

func (app *App) StartClient() *exec.Cmd {
	cmd := exec.Command(app.settings.ClientPath())
	cmd.Dir = app.settings.Client.Directory
	cmd.Start()
	return cmd
}

func (app *App) ShowSettings() {
	settings := app.NewWindow("Settings")
	settings.SetFixedSize(true)
	settings.Resize(fyne.NewSize(800, 600))
	settings.SetIcon(theme.StorageIcon())

	app.LoadSettingsContent(settings)
	settings.CenterOnScreen()
	settings.Show()
}

func (app *App) RequestPatch(version string, server *resource.Server) (resource.Update, error) {
	url, err := url.JoinPath(server.PatchServer, "patches", version)
	if err != nil {
		return resource.Update{}, fmt.Errorf("could not create patch version URL with \"%s\": %v", server.PatchServer, err)
	}

	response, err := http.Get(url)
	if err != nil {
		return resource.Update{}, ErrPatchesUnavailable
	}
	defer response.Body.Close()

	if response.StatusCode >= 300 {
		return resource.Update{}, ErrPatchesUnavailable
	}

	data, err := io.ReadAll(response.Body)
	if err != nil {
		return resource.Update{}, fmt.Errorf("cannot read body of patch version response: %v", err)
	}

	update := resource.Update{}
	err = json.Unmarshal(data, &update)
	if err != nil {
		return resource.Update{}, fmt.Errorf("malformed response body from patch version: %v", err)
	}

	return update, nil
}

func (app *App) Update(server *resource.Server) {
	app.SetUpdatingState()

	patches, ok := app.serverPatches[server.Id]
	if !ok {
		log.Printf("Patches missing for \"%s\"\n", server.Name)
		return
	}

	go func(version string, server *resource.Server) {
		log.Printf("Getting patch \"%s\" for %s\n", version, server.Name)
		update, err := app.RequestPatch(version, server)
		if err != nil {
			log.Printf("Patch error: %v\n", err)
			if err != ErrPatchesUnavailable {
				dialog.ShowError(err, app.main)
			}

			app.SetNormalState()
			return
		}

		log.Printf("Patch received with %d downloads\n", len(update.Download))

		log.Println("Starting update...")
		err = update.Run(server)
		if err != nil {
			dialog.ShowError(err, app.main)
		}
		log.Println("Update completed.")

		if app.CurrentServer().Id == server.Id {
			app.SetCurrentServerInfo(server)
		}

		server.CurrentPatch = version
		app.servers.SaveInfo()

		app.SetNormalState()
	}(patches.CurrentVersion, server)
}

func (app *App) RequestPatches(server *resource.Server) (resource.Patches, error) {
	url, err := url.JoinPath(server.PatchServer, "patches")
	if err != nil {
		return resource.Patches{}, fmt.Errorf("could not create patch server URL with \"%s\": %v", server.PatchServer, err)
	}

	response, err := http.Get(url)
	if err != nil {
		return resource.Patches{}, ErrPatchesUnavailable
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusServiceUnavailable {
		return resource.Patches{}, ErrPatchesUnsupported
	}

	if response.StatusCode >= 300 {
		return resource.Patches{}, fmt.Errorf("invalid response status code from patch server: %d", response.StatusCode)
	}

	data, err := io.ReadAll(response.Body)
	if err != nil {
		return resource.Patches{}, fmt.Errorf("cannot read body of patch server response: %v", err)
	}

	patches := resource.Patches{}
	err = json.Unmarshal(data, &patches)
	if err != nil {
		return resource.Patches{}, fmt.Errorf("malformed response body from patch server: %v", err)
	}

	return patches, nil
}

func (app *App) CheckForUpdates(server *resource.Server) {
	if len(server.PatchServer) == 0 {
		return
	}

	if _, ok := app.serverPatches[server.Id]; ok {
		log.Printf("Patches for \"%s\" already received", server.Name)
		return
	}

	app.SetCheckingUpdatesState()
	go func(server *resource.Server) {
		log.Printf("Checking for updates for \"%s\"\n", server.Name)
		patches, err := app.RequestPatches(server)
		if err != nil {
			log.Printf("Patch server error: %v\n", err)
			if err != ErrPatchesUnavailable && err != ErrPatchesUnsupported {
				dialog.ShowError(err, app.main)
			}

			app.serverPatches[server.Id] = resource.Patches{}
			app.SetNormalState()
			return
		}

		if server.CurrentPatch == patches.CurrentVersion {
			app.serverPatches[server.Id] = resource.Patches{}
			app.SetNormalState()
			return
		}

		log.Printf("Patch version \"%s\" is available\n", patches.CurrentVersion)

		app.serverPatches[server.Id] = patches
		app.SetUpdateState()
	}(server)
}

func (app *App) Start() {
	app.CheckForUpdates(app.CurrentServer())

	app.main.CenterOnScreen()
	app.main.ShowAndRun()
}
