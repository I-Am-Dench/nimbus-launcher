package app

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
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

	serverSelector     *widget.Select
	playButton         *widget.Button
	definiteProgress   *widget.ProgressBar
	indefiniteProgress *widget.ProgressBarInfinite

	progressText string

	serverNameBinding binding.String
	authServerBinding binding.String
	localeBinding     binding.String

	signupBinding binding.String
	signinBinding binding.String

	serverPatches map[string]resource.ServerPatches

	clientErrorIcon *widget.Icon
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

	a.settingsWindow = nil

	icon, err := resource.Asset(IMAGE_ICON)
	if err == nil {
		a.main.SetIcon(icon)
	} else {
		log.Println(fmt.Errorf("unable to load icon: %v", err))
	}

	a.clientErrorIcon = widget.NewIcon(theme.NewErrorThemedResource(theme.ErrorIcon()))
	a.clientErrorIcon.Hide()

	// log.Printf("Using \"%s\" as client directory\n", a.settings.Client.Directory)

	// _, err = os.Stat(a.settings.ClientPath())
	// if a.FoundClient = !errors.Is(err, os.ErrNotExist); a.FoundClient {
	// 	log.Printf("Found valid client \"%s\"\n", a.settings.Client.Name)
	// } else {
	// 	log.Printf("Cannot find valid executable \"%s\" in client directory: %v", a.settings.Client.Name, err)
	// }

	a.serverNameBinding = binding.NewString()
	a.authServerBinding = binding.NewString()
	a.localeBinding = binding.NewString()

	a.signupBinding = binding.NewString()
	a.signinBinding = binding.NewString()

	a.serverPatches = make(map[string]resource.ServerPatches)

	a.LoadContent()

	return a
}

func (app *App) SetCurrentServerInfo(server *resource.Server) {
	if server == nil {
		server = &resource.Server{}
		server.Config = &luconfig.LUConfig{}
	}

	app.serverNameBinding.Set(server.Config.ServerName)
	app.authServerBinding.Set(server.Config.AuthServerIP)
	app.localeBinding.Set(server.Config.Locale)

	app.signupBinding.Set(server.Config.SignupURL)
	app.signinBinding.Set(server.Config.SigninURL)
}

func (app *App) SetCurrentServer(server *resource.Server) {
	app.SetCurrentServerInfo(server)

	if server != nil {
		app.settings.SelectedServer = server.Id
	} else {
		app.settings.SelectedServer = ""
		app.serverSelector.ClearSelected()
	}

	err := app.settings.Save()
	if err != nil {
		log.Printf("save settings err: %v\n", err)
	}

	if app.IsReady() && app.settings.CheckPatchesAutomatically {
		app.CheckForUpdates(server)
	}
}

func (app *App) Refresh() {
	if server := app.CurrentServer(); server != nil {
		app.serverSelector.SetSelectedIndex(app.servers.Find(server.Id))
		return
	}
	app.SetCurrentServer(nil)
}

func (app *App) SetPlayingState() {
	app.HideProgress()

	app.playButton.Disable()
	app.playButton.SetText("Playing")

	app.serverSelector.Disable()
}

func (app *App) SetNormalState() {
	app.HideProgress()

	app.playButton.Enable()
	app.playButton.SetText("Play")
	app.playButton.SetIcon(theme.MediaPlayIcon())
	app.playButton.Importance = widget.HighImportance
	app.playButton.OnTapped = app.PressPlay
	app.playButton.Refresh()

	app.serverSelector.Enable()
}

func (app *App) SetUpdatingState() {
	app.ShowIndefiniteProgress()

	app.playButton.Disable()
	app.playButton.SetText("Updating")
}

func (app *App) SetUpdateState() {
	app.HideProgress()

	app.playButton.Enable()
	app.playButton.SetText("Update")
	app.playButton.SetIcon(theme.DownloadIcon())
	app.playButton.Importance = widget.SuccessImportance
	app.playButton.OnTapped = app.PressUpdate
	app.playButton.Refresh()
}

func (app *App) SetCheckingUpdatesState() {
	app.ShowIndefiniteProgress()

	app.playButton.Disable()
	app.playButton.SetText("Checking updates")
	app.playButton.SetIcon(nil)
	app.playButton.Refresh()
}

func (app *App) ShowIndefiniteProgress() {
	app.definiteProgress.Hide()
	app.indefiniteProgress.Show()
}

func (app *App) ShowProgress(value float64, format string) {
	app.definiteProgress.SetValue(value)
	app.progressText = format

	app.indefiniteProgress.Hide()
	app.definiteProgress.Show()
}

func (app *App) HideProgress() {
	app.definiteProgress.Hide()
	app.indefiniteProgress.Hide()
}

func (app *App) CurrentServer() *resource.Server {
	return app.servers.Get(app.settings.SelectedServer)
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

	if app.settings.SelectedServer != app.settings.PreviouslyRunServer {
		log.Println("Selected server does not match previously run server; Copying over boot.cfg")
		err := app.CopyBootConfiguration(app.CurrentServer())
		if err != nil {
			dialog.ShowError(fmt.Errorf("could not copy \"boot.cfg\": %v", err), app.main)
			app.SetNormalState()
			return
		}
		log.Println("Copy completed.")
	}

	app.settings.PreviouslyRunServer = app.settings.SelectedServer
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
	if len(strings.TrimSpace(app.settings.Client.RunCommand)) > 0 {
		cmd = exec.Command(app.settings.Client.RunCommand, app.settings.ClientPath())
	}

	cmd.Dir = app.settings.Client.Directory
	cmd.Env = strings.Split(app.settings.Client.EnvironmentVariables, ";")
	cmd.Start()
	return cmd
}

func (app *App) ShowSettings() {
	if app.settingsWindow != nil {
		app.settingsWindow.RequestFocus()
		return
	}

	app.settings.PreviouslyRunServer = ""
	app.settings.Save()

	settings := app.NewWindow("Settings")
	settings.SetFixedSize(true)
	settings.Resize(fyne.NewSize(800, 600))
	settings.SetIcon(theme.StorageIcon())
	settings.SetOnClosed(func() {
		app.settingsWindow = nil
		app.serverSelector.Enable()
	})

	app.serverSelector.Disable()

	app.LoadSettingsContent(settings)
	app.settingsWindow = settings

	settings.CenterOnScreen()
	settings.Show()
}

// func (app *App) RequestPatch(version string, server *resource.Server) (resource.Update, error) {
// 	url, err := url.JoinPath(server.PatchServer, "patches", version)
// 	if err != nil {
// 		return resource.Update{}, fmt.Errorf("could not create patch version URL with \"%s\": %v", server.PatchServer, err)
// 	}

// 	response, err := http.Get(url)
// 	if err != nil {
// 		return resource.Update{}, ErrPatchesUnavailable
// 	}
// 	defer response.Body.Close()

// 	if response.StatusCode >= 300 {
// 		return resource.Update{}, ErrPatchesUnavailable
// 	}

// 	data, err := io.ReadAll(response.Body)
// 	if err != nil {
// 		return resource.Update{}, fmt.Errorf("cannot read body of patch version response: %v", err)
// 	}

// 	update := resource.Update{}
// 	err = json.Unmarshal(data, &update)
// 	if err != nil {
// 		return resource.Update{}, fmt.Errorf("malformed response body from patch version: %v", err)
// 	}

// 	return update, nil
// }

// func (app *App) Update(server *resource.Server) {
// 	app.SetUpdatingState()

// 	patches, ok := app.serverPatches[server.Id]
// 	if !ok {
// 		log.Printf("Patches missing for \"%s\"\n", server.Name)
// 		return
// 	}

// 	go func(version string, server *resource.Server) {
// 		log.Printf("Getting patch \"%s\" for %s\n", version, server.Name)
// 		update, err := app.RequestPatch(version, server)
// 		if err != nil {
// 			log.Printf("Patch error: %v\n", err)
// 			if err != ErrPatchesUnavailable {
// 				dialog.ShowError(err, app.main)
// 			}

// 			app.SetNormalState()
// 			return
// 		}

// 		log.Printf("Patch received with %d downloads\n", len(update.Download))

// 		log.Println("Starting update...")
// 		err = update.Run(server)
// 		if err != nil {
// 			dialog.ShowError(err, app.main)
// 		}
// 		log.Println("Update completed.")

// 		if app.CurrentServer().Id == server.Id {
// 			app.SetCurrentServerInfo(server)
// 		}

// 		server.CurrentPatch = version
// 		app.servers.SaveInfos()

// 		app.SetNormalState()
// 	}(patches.CurrentVersion, server)
// }

func (app *App) Update(server *resource.Server) {
	app.SetUpdateState()

	patches, ok := app.serverPatches[server.Id]
	if !ok {
		log.Printf("Patches missing for \"%s\"\n", server.Name)
		return
	}

	go func(version string, server *resource.Server) {
		log.Printf("Getting patch \"%s\" for %s\n", version, server.Name)
		patch, err := resource.GetPatch(version, server)
		if err != nil {
			log.Printf("Patch error: %v\n", err)
			if err != resource.ErrPatchesUnavailable {
				dialog.ShowError(err, app.main)
			}

			app.SetNormalState()
			return
		}

		log.Printf("Patch received with %d downloads\n", len(patch.Downloads))

		log.Println("Starting update...")
		err = patch.RunWithDependencies(server)
		if err != nil {
			dialog.ShowError(err, app.main)
		}
		log.Println("Update completed.")

		app.Refresh()

		server.CurrentPatch = version
		app.servers.SaveInfos()

		app.SetNormalState()
	}(patches.CurrentVersion, server)
}

// func (app *App) RequestPatches(server *resource.Server) (resource.Patches, error) {
// 	url, err := url.JoinPath(server.PatchServer, "patches")
// 	if err != nil {
// 		return resource.Patches{}, fmt.Errorf("could not create patch server URL with \"%s\": %v", server.PatchServer, err)
// 	}

// 	response, err := http.Get(url)
// 	if err != nil {
// 		return resource.Patches{}, ErrPatchesUnavailable
// 	}
// 	defer response.Body.Close()

// 	if response.StatusCode == http.StatusServiceUnavailable {
// 		return resource.Patches{}, ErrPatchesUnsupported
// 	}

// 	if response.StatusCode >= 300 {
// 		return resource.Patches{}, fmt.Errorf("invalid response status code from patch server: %d", response.StatusCode)
// 	}

// 	data, err := io.ReadAll(response.Body)
// 	if err != nil {
// 		return resource.Patches{}, fmt.Errorf("cannot read body of patch server response: %v", err)
// 	}

// 	patches := resource.Patches{}
// 	err = json.Unmarshal(data, &patches)
// 	if err != nil {
// 		return resource.Patches{}, fmt.Errorf("malformed response body from patch server: %v", err)
// 	}

// 	return patches, nil
// }

// func (app *App) CheckForUpdates(server *resource.Server) {
// 	if server == nil {
// 		return
// 	}

// 	if len(server.PatchServer) == 0 || !app.clientErrorIcon.Hidden {
// 		return
// 	}

// 	if _, ok := app.serverPatches[server.Id]; ok {
// 		log.Printf("Patches for \"%s\" already received", server.Name)
// 		return
// 	}

// 	app.SetCheckingUpdatesState()
// 	go func(server *resource.Server) {
// 		log.Printf("Checking for updates for \"%s\"\n", server.Name)
// 		patches, err := app.RequestPatches(server)
// 		if err != nil {
// 			log.Printf("Patch server error: %v\n", err)
// 			if err != ErrPatchesUnavailable && err != ErrPatchesUnsupported {
// 				dialog.ShowError(err, app.main)
// 			}

// 			app.serverPatches[server.Id] = resource.Patches{}
// 			app.SetNormalState()
// 			return
// 		}

// 		if server.CurrentPatch == patches.CurrentVersion {
// 			app.serverPatches[server.Id] = resource.Patches{}
// 			app.SetNormalState()
// 			return
// 		}

// 		log.Printf("Patch version \"%s\" is available\n", patches.CurrentVersion)

// 		app.serverPatches[server.Id] = patches
// 		app.SetUpdateState()
// 	}(server)
// }

func (app *App) CheckForUpdates(server *resource.Server) {
	if server == nil {
		return
	}

	if len(server.Config.PatchServerIP) == 0 || !app.clientErrorIcon.Hidden {
		return
	}

	if _, ok := app.serverPatches[server.Id]; ok {
		log.Printf("Patches for \"%s\" already received", server.Name)
		return
	}

	app.SetCheckingUpdatesState()
	go func(server *resource.Server) {
		log.Printf("Checking for updates for \"%s\"; Current version: \"%s\"\n", server.Name, server.CurrentPatch)
		patches, err := resource.GetServerPatches(server)
		if err != nil {
			log.Printf("Patch server error: %v\n", err)
			if err != resource.ErrPatchesUnavailable && err != resource.ErrPatchesUnsupported {
				dialog.ShowError(err, app.main)
			}

			if err != resource.ErrPatchesUnauthorized {
				app.serverPatches[server.Id] = resource.ServerPatches{}
			}

			app.SetNormalState()
			return
		}

		if server.CurrentPatch == patches.CurrentVersion {
			log.Println("Server is already latest version.")
			app.serverPatches[server.Id] = patches
			app.SetNormalState()
			return
		}

		log.Printf("Patch version \"%s\" is available\n", patches.CurrentVersion)

		app.serverPatches[server.Id] = patches
		app.SetUpdateState()
	}(server)
}

func (app *App) IsReady() bool {
	return app.playButton != nil
}

// This functionality can be expanded upon later
func (app *App) IsValidClient(path string) (bool, error) {
	stats, err := os.Stat(path)
	return !errors.Is(err, os.ErrNotExist) && !stats.IsDir(), err
}

func (app *App) CheckClient() {
	log.Printf("Using \"%s\" as client directory\n", app.settings.Client.Directory)

	if ok, err := app.IsValidClient(app.settings.ClientPath()); ok {
		log.Printf("Found valid client \"%s\"\n", app.settings.Client.Name)
		app.clientErrorIcon.Hide()
		app.SetNormalState()
	} else {
		log.Printf("Cannot find valid executable \"%s\" in client directory: %v", app.settings.Client.Name, err)
		app.playButton.Disable()
		app.clientErrorIcon.Show()
	}
}

func (app *App) Start() {
	app.CheckClient()

	if app.settings.CheckPatchesAutomatically {
		app.CheckForUpdates(app.CurrentServer())
	}

	app.main.CenterOnScreen()
	app.main.ShowAndRun()
}
