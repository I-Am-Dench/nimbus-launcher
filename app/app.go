package app

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/I-Am-Dench/lu-launcher/clientcache"
	"github.com/I-Am-Dench/lu-launcher/luconfig"
	"github.com/I-Am-Dench/lu-launcher/resource"
)

type App struct {
	fyne.App
	settings    *resource.Settings
	servers     resource.ServerList
	clientCache clientcache.ClientCache

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

func New(settings *resource.Settings, servers resource.ServerList) App {
	a := App{}
	a.App = app.New()

	a.settings = settings
	a.servers = servers

	cache, err := resource.ClientCache()
	if err != nil {
		log.Panicf("Could not create client cache database: %v", err)
	}
	a.clientCache = cache

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
	server := app.CurrentServer()
	if server == nil {
		app.SetCurrentServer(nil)
		return
	}

	app.serverSelector.SetSelectedIndex(app.servers.Find(server.Id))
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

func (app *App) SetProgress(value float64) {
	app.ShowProgress(value, app.progressText)
}

func (app *App) HideProgress() {
	app.definiteProgress.Hide()
	app.indefiniteProgress.Hide()
}

func (app *App) CurrentServer() *resource.Server {
	return app.servers.Get(app.settings.SelectedServer)
}

func (app *App) TransferCachedClientResources() error {
	log.Println("Transferring cached client resources...")

	resources, err := app.clientCache.GetResources()
	if err != nil {
		return fmt.Errorf("could not query resources")
	}
	log.Printf("Queried %d cached resources.", len(resources))

	app.ShowProgress(0, "Transferring resources: %f%%")
	defer app.HideProgress()
	for i, resource := range resources {
		log.Printf("Transferring cached resource: %s\n", resource.Path)
		err := clientcache.WriteResource(app.settings.Client.Directory, resource)
		if err != nil {
			return fmt.Errorf("could not transfer cached resource")
		}
		app.SetProgress(float64((i + 1) / len(resources)))
	}

	log.Println("Completed transfer(s)!")
	return nil
}

func (app *App) TransferPatchResources(server *resource.Server) error {
	log.Println("Transfer patch resources...")
	patch, err := resource.GetPatch(server.CurrentPatch, server)
	if err != nil {
		return err
	}

	err = patch.TransferResources(app.settings.Client.Directory, app.clientCache, server)
	if err != nil {
		return err
	}

	log.Println("Completed transfer(s)!")
	return nil
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

	server := app.CurrentServer()
	log.Printf("Selected server: %s\n", server.Name)

	err := app.TransferCachedClientResources()
	if err != nil {
		log.Println(err)
		dialog.ShowError(fmt.Errorf("client resources may be incorrect when running: %v", err), app.main)
	}

	if app.settings.SelectedServer != app.settings.PreviouslyRunServer {
		log.Println("Selected server does not match previously run server; Copying over boot.cfg")
		err := app.CopyBootConfiguration(server)
		if err != nil {
			dialog.ShowError(fmt.Errorf("could not copy \"boot.cfg\": %v", err), app.main)
			app.SetNormalState()
			return
		}
		log.Println("Copy completed.")
	}

	if len(server.CurrentPatch) > 0 {
		err := app.TransferPatchResources(server)
		if err != nil {
			log.Println(err)
			dialog.ShowError(fmt.Errorf("patch resources may be incorrect when running: %v", err), app.main)
		}
	}

	app.settings.PreviouslyRunServer = app.settings.SelectedServer
	app.settings.Save()

	log.Println("Launching Lego Universe...")
	log.Printf("Close launcher when played: %v\n", app.settings.CloseOnPlay)

	go func() {
		time.Sleep(2 * time.Second)
		app.SetNormalState()
	}()

	// cmd := app.StartClient()
	// if app.settings.CloseOnPlay {
	// 	app.main.Close()
	// 	return
	// }

	// go func(cmd *exec.Cmd) {
	// 	cmd.Wait()
	// 	app.SetNormalState()
	// }(cmd)
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
