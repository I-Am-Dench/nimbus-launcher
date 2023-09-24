package app

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/I-Am-Dench/lu-launcher/client"
	"github.com/I-Am-Dench/lu-launcher/luconfig"
	"github.com/I-Am-Dench/lu-launcher/luwidgets"
	"github.com/I-Am-Dench/lu-launcher/resource"
)

type App struct {
	fyne.App
	settings        *resource.Settings
	rejectedPatches resource.RejectedPatches

	client client.Client

	clientCache client.Cache

	main           fyne.Window
	settingsWindow fyne.Window
	patchWindow    fyne.Window

	serverList *luwidgets.ServerList

	playButton           *widget.Button
	refreshUpdatesButton *widget.Button
	progressBar          *luwidgets.BinaryProgressBar

	serverNameBinding binding.String
	authServerBinding binding.String
	localeBinding     binding.String

	signupBinding binding.String
	signinBinding binding.String

	clientErrorIcon *widget.Icon
}

func New(settings *resource.Settings, servers resource.ServerList, rejectedPatches resource.RejectedPatches) App {
	a := App{}
	a.App = app.New()

	a.settings = settings
	a.rejectedPatches = rejectedPatches

	a.client = client.NewStandardClient()

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

	a.serverNameBinding = binding.NewString()
	a.authServerBinding = binding.NewString()
	a.localeBinding = binding.NewString()

	a.signupBinding = binding.NewString()
	a.signinBinding = binding.NewString()

	a.InitializeGlobalWidgets(servers)

	a.LoadContent()

	return a
}

func (app *App) InitializeGlobalWidgets(servers resource.ServerList) {
	app.clientErrorIcon = widget.NewIcon(theme.NewErrorThemedResource(theme.ErrorIcon()))
	app.clientErrorIcon.Hide()

	app.refreshUpdatesButton = widget.NewButtonWithIcon(
		"Check For Updates", theme.ViewRefreshIcon(),
		func() {
			app.CheckForUpdates(app.CurrentServer())
		},
	)

	app.playButton = widget.NewButtonWithIcon(
		"Play", theme.MediaPlayIcon(),
		app.PressPlay,
	)
	app.playButton.Importance = widget.HighImportance

	app.progressBar = luwidgets.NewBinaryProgressBar()

	app.serverList = luwidgets.NewServerList(servers, app.OnServerChanged)
	app.serverList.SetSelectedServer(app.settings.SelectedServer)
}

func (app *App) BindServerInfo(server *resource.Server) {
	if server == nil {
		server = &resource.Server{}
		server.Config = &luconfig.LUConfig{}
	}

	app.serverNameBinding.Set(server.Config.ServerName)
	app.authServerBinding.Set(server.Config.AuthServerIP)
	app.localeBinding.Set(server.Config.Locale)

	app.signupBinding.Set(server.Config.SignupURL)
	app.signinBinding.Set(server.Config.SigninURL)

	if len(server.Config.PatchServerIP) > 0 {
		app.refreshUpdatesButton.Show()
	} else {
		app.refreshUpdatesButton.Hide()
	}
}

func (app *App) OnServerChanged(server *resource.Server) {
	app.BindServerInfo(server)

	if server != nil {
		app.settings.SelectedServer = server.Id
	} else {
		app.settings.SelectedServer = ""
	}

	err := app.settings.Save()
	if err != nil {
		log.Printf("save settings error: %v\n", err)
	}

	if app.IsReady() && app.settings.CheckPatchesAutomatically {
		app.CheckForUpdates(server)
	} else if server != nil {
		if server.PendingUpdate() {
			app.SetUpdateState()
		} else {
			app.SetNormalState()
		}
	}
}

func (app *App) CurrentServer() *resource.Server {
	return app.serverList.SelectedServer()
}

func (app *App) TransferCachedClientResources() error {
	log.Println("Transferring cached client resources...")

	resources, err := app.clientCache.GetResources()
	if err != nil {
		return fmt.Errorf("could not query resources")
	}
	log.Printf("Queried %d cached resources.", len(resources))

	app.progressBar.SetMax(float64(len(resources)))
	app.progressBar.ShowValue(0, "Transferring resources: $VALUE/$MAX")
	defer app.progressBar.Hide()
	for i, resource := range resources {
		log.Printf("Transferring cached resource: %s\n", resource.Path)
		err := client.WriteResource(app.settings.Client.Directory, resource)
		if err != nil {
			return fmt.Errorf("could not transfer cached resource")
		}

		app.progressBar.SetValue(float64(i + 1))
	}

	app.progressBar.ShowFormat("Completed transfer(s)!")
	log.Println("Completed transfer(s)!")
	return nil
}

func (app *App) TransferPatchResources(server *resource.Server) error {
	log.Println("Transfer patch resources...")
	patch, err := resource.GetPatch(server.CurrentPatch, server)
	if err != nil {
		return err
	}

	app.progressBar.ShowIndefinite()
	err = patch.TransferResources(app.settings.Client.Directory, app.clientCache, server)
	if err != nil {
		return err
	}

	app.progressBar.ShowFormat("Completed transfer(s)!")
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
	if server == nil {
		dialog.ShowInformation("Select Server", "Please select a server.", app.main)
		app.SetNormalState()
		return
	}

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

	cmd, err := app.client.Start()
	if err != nil {
		log.Println(err)
		dialog.ShowError(err, app.main)
		app.SetNormalState()
		return
	}

	if app.settings.CloseOnPlay {
		app.main.Close()
		return
	}

	// cmd := app.StartClient()
	// if app.settings.CloseOnPlay {
	// 	app.main.Close()
	// 	return
	// }

	app.progressBar.Hide()
	go func(cmd *exec.Cmd) {
		cmd.Wait()
		app.SetNormalState()
	}(cmd)
}

func (app *App) PressUpdate() {
	app.Update(app.CurrentServer())
}

// func (app *App) StartClient() *exec.Cmd {
// 	cmd := exec.Command(app.settings.ClientPath())
// 	if len(strings.TrimSpace(app.settings.Client.RunCommand)) > 0 {
// 		cmd = exec.Command(app.settings.Client.RunCommand, app.settings.ClientPath())
// 	}

// 	cmd.Dir = app.settings.Client.Directory
// 	cmd.Env = strings.Split(app.settings.Client.EnvironmentVariables, ";")
// 	cmd.Start()
// 	return cmd
// }

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

		if app.client.IsValid() {
			app.serverList.Enable()
		}
	})

	app.serverList.Disable()

	app.LoadSettingsContent(settings)
	app.settingsWindow = settings

	settings.CenterOnScreen()
	settings.Show()
}

func (app *App) ShowPatch(patch resource.Patch, onConfirmCancel func(PatchAcceptState)) {
	if app.patchWindow != nil {
		app.patchWindow.RequestFocus()
		return
	}

	window := app.NewWindow("Review Patch")
	window.SetFixedSize(true)
	window.Resize(fyne.NewSize(800, 600))
	window.SetIcon(theme.QuestionIcon())
	window.SetOnClosed(func() {
		app.patchWindow = nil
		onConfirmCancel(PatchCancel)
	})

	app.LoadPatchContent(window, patch, onConfirmCancel)
	app.patchWindow = window

	app.patchWindow.CenterOnScreen()
	app.patchWindow.Show()
}

func (app *App) RunUpdate(server *resource.Server, patch resource.Patch) {
	defer app.serverList.RemoveAsUpdating(server)

	log.Println("Starting update...")
	err := patch.RunWithDependencies(server)
	if err != nil {
		log.Println(err)
		dialog.ShowError(err, app.main)
		return
	}
	log.Println("Update completed.")

	app.serverList.Refresh()

	server.CurrentPatch = patch.Version
	app.serverList.Save()
}

func (app *App) Update(server *resource.Server) {
	app.SetUpdatingState()

	app.serverList.MarkAsUpdating(server)

	patches, ok := server.ServerPatches()
	if !ok {
		log.Printf("Patches missing for \"%s\"\n", server.Name)
		return
	}

	go func(version string, server *resource.Server) {
		defer server.SetPendingUpdate(false)
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

		if !app.settings.ReviewPatchBeforeUpdate {
			app.RunUpdate(server, patch)
			app.SetNormalState()
			return
		}

		app.ShowPatch(patch, func(state PatchAcceptState) {
			defer app.SetNormalState()

			if state == PatchCancel {
				return
			}

			if state == PatchReject {
				err := app.rejectedPatches.Add(server, patch.Version)
				if err == nil {
					log.Printf("Rejected patch version \"%s\"\n", patch.Version)
				} else {
					dialog.ShowError(fmt.Errorf("failed to reject patch: %v", err), app.main)
				}
				return
			}

			app.RunUpdate(server, patch)
		})
	}(patches.CurrentVersion, server)
}

func (app *App) CheckForUpdates(server *resource.Server) {
	if server == nil {
		return
	}

	if len(server.Config.PatchServerIP) == 0 || !app.clientErrorIcon.Hidden {
		return
	}

	if _, ok := server.ServerPatches(); ok {
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
				server.SetServerPatches(resource.ServerPatches{})
			}

			app.SetNormalState()
			return
		}

		if server.CurrentPatch == patches.CurrentVersion {
			log.Println("Server is already latest version.")
			server.SetServerPatches(patches)
			app.SetNormalState()
			return
		}

		if err := resource.ValidateVersionName(patches.CurrentVersion); err != nil {
			log.Println(err)
			server.SetServerPatches(patches)
			app.SetNormalState()
			return
		}

		log.Printf("Patch version \"%s\" is available\n", patches.CurrentVersion)

		if app.rejectedPatches.IsRejected(server, patches.CurrentVersion) {
			log.Printf("Patch version \"%s\" is rejected; Aborting update sequence.\n", patches.CurrentVersion)
			app.SetNormalState()
			return
		}

		server.SetServerPatches(patches)
		server.SetPendingUpdate(true)
		app.SetUpdateState()
	}(server)
}

func (app *App) IsReady() bool {
	return app.playButton != nil
}

// This functionality can be expanded upon later
// func (app *App) IsValidClient(path string) (bool, error) {
// 	stats, err := os.Stat(path)
// 	return !errors.Is(err, os.ErrNotExist) && !stats.IsDir(), err
// }

func (app *App) CheckClient() {
	log.Printf("Using \"%s\" as client directory\n", app.settings.Client.Directory)

	err := app.client.SetPath(app.settings.ClientPath())
	if err != nil {
		log.Printf("Cannot find executable \"%s\" in client directory: %v", app.settings.Client.Name, err)
		app.playButton.Disable()
		app.serverList.Disable()
		app.clientErrorIcon.Show()
	} else {
		log.Printf("Found valid client \"%s\"\n", app.settings.Client.Name)
		app.clientErrorIcon.Hide()
		app.serverList.Enable()
		app.SetNormalState()
	}
}

// func (app *App) CheckClient() {
// 	log.Printf("Using \"%s\" as client directory\n", app.settings.Client.Directory)

// 	if ok, err := app.IsValidClient(app.settings.ClientPath()); ok {
// 		log.Printf("Found valid client \"%s\"\n", app.settings.Client.Name)
// 		app.clientErrorIcon.Hide()
// 		app.SetNormalState()
// 	} else {
// 		log.Printf("Cannot find valid executable \"%s\" in client directory: %v", app.settings.Client.Name, err)
// 		app.playButton.Disable()
// 		app.clientErrorIcon.Show()
// 	}
// }

func (app *App) Start() {
	app.CheckClient()

	if app.settings.CheckPatchesAutomatically {
		app.CheckForUpdates(app.CurrentServer())
	}

	app.main.CenterOnScreen()
	app.main.ShowAndRun()
}
