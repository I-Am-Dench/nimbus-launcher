package app

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/I-Am-Dench/nimbus-launcher/app/multiwindow"
	"github.com/I-Am-Dench/nimbus-launcher/app/nlwidgets"
	"github.com/I-Am-Dench/nimbus-launcher/app/nlwindows"
	"github.com/I-Am-Dench/nimbus-launcher/app/nlwindows/info"
	"github.com/I-Am-Dench/nimbus-launcher/app/nlwindows/patchreview"
	"github.com/I-Am-Dench/nimbus-launcher/app/nlwindows/prerequisites"
	wsettings "github.com/I-Am-Dench/nimbus-launcher/app/nlwindows/settings"
	"github.com/I-Am-Dench/nimbus-launcher/client"
	"github.com/I-Am-Dench/nimbus-launcher/ldf"
	"github.com/I-Am-Dench/nimbus-launcher/resource"
	"github.com/I-Am-Dench/nimbus-launcher/resource/patch"
	"github.com/I-Am-Dench/nimbus-launcher/resource/server"
	"github.com/I-Am-Dench/nimbus-launcher/version"
)

type App struct {
	*multiwindow.App

	settings        *resource.Settings
	rejectedPatches *patch.RejectionList

	client client.Client

	clientResources client.Resources

	main        fyne.Window
	patchWindow fyne.Window

	serverList *nlwidgets.ServerList

	playButton           *widget.Button
	refreshUpdatesButton *widget.Button
	progressBar          *nlwidgets.BinaryProgressBar

	serverNameBinding binding.String
	authServerBinding binding.String
	localeBinding     binding.String

	clientPathBinding binding.String

	signupBinding binding.String
	signinBinding binding.String

	clientErrorIcon *widget.Icon
}

func New(settings *resource.Settings, servers resource.ServerList, rejectedPatches *patch.RejectionList) App {
	a := App{}
	a.App = multiwindow.New()

	a.settings = settings
	a.rejectedPatches = rejectedPatches

	a.client = client.NewStandardClient()

	resources, err := resource.ClientResources()
	if err != nil {
		log.Panicf("Could not create client cache database: %v", err)
	}
	a.clientResources = resources

	a.main = a.NewWindow(fmt.Sprintf("Nimbus Launcher (%v)", version.Get().Name()))
	a.main.SetFixedSize(true)
	a.main.Resize(fyne.NewSize(800, 300))
	a.main.SetMaster()

	icon := resource.Icon()
	if err == nil {
		a.main.SetIcon(icon)
	} else {
		log.Println(fmt.Errorf("unable to load icon: %v", err))
	}

	a.serverNameBinding = binding.NewString()
	a.authServerBinding = binding.NewString()
	a.localeBinding = binding.NewString()

	a.clientPathBinding = binding.NewString()

	a.signupBinding = binding.NewString()
	a.signinBinding = binding.NewString()

	a.InitializeGlobalWidgets(servers)

	a.LoadContent()

	a.main.SetOnClosed(func() {
		err := a.clientResources.Close()
		if err != nil {
			log.Printf("could not properly close clientCache: %v", err)
		}
	})

	info.New(a.App, nlwindows.Info)

	wsettings.New(a.App, nlwindows.Settings, a.serverList, a.settings, func() {
		a.CheckClient()
	}).SetOnClosed(func() {
		if a.client.IsValid() {
			a.serverList.Enable()
		}
	})

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

	app.progressBar = nlwidgets.NewBinaryProgressBar()

	app.serverList = nlwidgets.NewServerList(servers, app.OnServerChanged)
	app.serverList.SetSelectedServer(app.settings.SelectedServer)
}

func (app *App) BindServerInfo(serv *server.Server) {
	if serv == nil {
		serv = &server.Server{}
		serv.Config = &ldf.BootConfig{}
	}

	app.serverNameBinding.Set(serv.Config.ServerName)
	app.authServerBinding.Set(serv.Config.AuthServerIP)
	app.localeBinding.Set(serv.Config.Locale)

	app.signupBinding.Set(serv.Config.SignupURL)
	app.signinBinding.Set(serv.Config.SigninURL)

	if len(serv.PatchProtocol) > 0 {
		app.refreshUpdatesButton.Show()
	} else {
		app.refreshUpdatesButton.Hide()
	}
}

func (app *App) OnServerChanged(server *server.Server) {
	app.BindServerInfo(server)

	if server != nil {
		app.settings.SelectedServer = server.ID
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
		if server.CheckingUpdates() {
			app.SetCheckingUpdatesState()
		} else if server.PendingUpdate() {
			app.SetUpdateState()
		} else {
			app.SetNormalState()
		}
	}
}

func (app *App) CurrentServer() *server.Server {
	return app.serverList.SelectedServer()
}

func (app *App) TransferCachedClientResources() error {
	defer app.progressBar.Hide()
	log.Println("Transferring cached client resources...")

	// Reset replaced resources
	replaced, err := app.clientResources.Replacements().List()
	if err != nil {
		return fmt.Errorf("could not query replaced resources: %w", err)
	}
	log.Printf("Queried %d replaced resources.", len(replaced))

	app.progressBar.SetMax(float64(len(replaced)))
	app.progressBar.ShowValue(0, "Transferring resources: $VALUE/$MAX")
	for i, resource := range replaced {
		log.Printf("Transferring replaced resource: %s", resource.Path)

		err := client.WriteResource(app.settings.Client.Directory, resource)
		if err != nil {
			return fmt.Errorf("could not transfer replaced resource: %w", err)
		}

		app.progressBar.SetValue(float64(i + 1))
	}

	// Delete added resources
	added, err := app.clientResources.Additions().List()
	if err != nil {
		return fmt.Errorf("could not query added resources: %w", err)
	}
	log.Printf("Queried %d added resources.", len(added))

	app.progressBar.SetMax(float64(len(replaced)))
	app.progressBar.ShowValue(0, "Removing added resources: $VALUE/$MAX")
	for i, resource := range added {
		log.Printf("Deleting added resource: %s", resource)

		err := client.RemoveResource(app.settings.Client.Directory, resource)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("could not remove added resource: %w", err)
		}

		app.progressBar.SetValue(float64(i + 1))
	}

	app.progressBar.ShowFormat("Completed transfer(s)!")
	log.Println("Completed transfer(s)!")
	return nil
}

func (app *App) TransferPatchResources(server *server.Server) error {
	log.Println("Transfer patch resources...")
	patch, err := server.GetPatch(server.CurrentPatch)
	if err != nil {
		return err
	}

	app.progressBar.ShowIndefinite()
	err = patch.TransferResourcesWithDependencies(app.settings.Client.Directory, app.clientResources, server)
	if err != nil {
		return err
	}

	app.progressBar.ShowFormat("Completed transfer(s)!")
	log.Println("Completed transfer(s)!")
	return nil
}

func (app *App) CopyBootConfiguration(server *server.Server) error {
	data, err := os.ReadFile(server.BootPath())
	if err != nil {
		return fmt.Errorf("cannot read \"%s\": %w", server.BootPath(), err)
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

	app.progressBar.Hide()
	go func(cmd *exec.Cmd) {
		if err := cmd.Wait(); err != nil {
			log.Println(err)
		}
		log.Println("Client exited.")
		app.SetNormalState()
	}(cmd)
}

func (app *App) PressUpdate() {
	app.Update(app.CurrentServer())
}

func (app *App) ShowSettings() {
	app.settings.PreviouslyRunServer = ""
	app.settings.Save()

	app.serverList.Disable()

	app.ShowInstanceWindow(nlwindows.Settings)
}

func (app *App) ShowPatch(patch patch.Patch, onConfirmCancel func(patchreview.Answer)) {
	if app.patchWindow != nil {
		app.patchWindow.RequestFocus()
		return
	}

	app.patchWindow = patchreview.New(app, patch, onConfirmCancel)
	app.patchWindow.SetOnClosed(func() {
		app.patchWindow = nil
		onConfirmCancel(patchreview.PatchAccept)
	})

	app.patchWindow.CenterOnScreen()
	app.patchWindow.Show()
}

func (app *App) ShowInfo() {
	app.ShowInstanceWindow(nlwindows.Info)
}

func (app *App) RunUpdate(server *server.Server, patch patch.Patch) {
	defer app.serverList.RemoveAsUpdating(server)

	log.Println("Starting update...")
	err := patch.UpdateResources(server, app.rejectedPatches)
	if err != nil {
		log.Println(err)
		dialog.ShowError(err, app.main)
		return
	}
	log.Println("Update completed.")

	app.serverList.Refresh()

	server.CurrentPatch = patch.Version()
	app.serverList.Save()
}

func (app *App) Update(serv *server.Server) {
	app.SetUpdatingState()

	app.serverList.MarkAsUpdating(serv)

	versions, ok := serv.PatchesSummary()
	if !ok {
		log.Printf("Patches missing for \"%s\"\n", serv.Name)
		return
	}

	go func(version string, serv *server.Server) {
		defer serv.SetState(server.Normal)
		log.Printf("Getting patch \"%s\" for %s\n", version, serv.Name)

		p, err := serv.GetPatch(version)
		if err != nil {
			log.Printf("Patch error: %v", err)
			if !errors.Is(err, patch.ErrPatchesUnavailable) {
				dialog.ShowError(err, app.main)
			}

			app.SetNormalState()
			return
		}

		log.Printf("Patch received: %s", p.Summary())

		if !app.settings.ReviewPatchBeforeUpdate {
			app.RunUpdate(serv, p)
			app.SetNormalState()
			return
		}

		app.ShowPatch(p, func(state patchreview.Answer) {
			defer app.SetNormalState()

			if state == patchreview.PatchCancel {
				return
			}

			if state == patchreview.PatchReject {
				err := app.rejectedPatches.Add(serv, p.Version())
				if err == nil {
					log.Printf("Rejected patch version \"%s\"\n", p.Version())
				} else {
					dialog.ShowError(fmt.Errorf("failed to reject patch: %v", err), app.main)
				}
				return
			}

			app.RunUpdate(serv, p)
		})
	}(versions.CurrentVersion, serv)
}

func (app *App) CheckForUpdates(serv *server.Server) {
	if serv == nil {
		return
	}

	if len(serv.Config.PatchServerIP) == 0 || !app.clientErrorIcon.Hidden {
		return
	}

	if _, ok := serv.PatchesSummary(); ok {
		log.Printf("Patches for \"%s\" already received", serv.Name)
		return
	}

	serv.SetState(server.CheckingUpdates)
	app.SetCheckingUpdatesState()
	go func(serv *server.Server) {
		log.Printf("Checking for updates for \"%s\"; Current version: \"%s\"\n", serv.Name, serv.CurrentPatch)

		patches, err := serv.GetPatchesSummary()
		if err != nil {
			log.Printf("Patch server error: %v\n", err)
			if err != patch.ErrPatchesUnavailable && err != patch.ErrPatchesUnsupported {
				dialog.ShowError(err, app.main)
			}

			if err != patch.ErrPatchesUnauthorized {
				serv.SetPatchesSummary(patch.Summary{})
			}

			serv.SetState(server.Normal)
			app.SetNormalState()
			return
		}

		if serv.CurrentPatch == patches.CurrentVersion {
			log.Println("Server is already latest version.")
			serv.SetPatchesSummary(patches)
			serv.SetState(server.Normal)
			app.SetNormalState()
			return
		}

		if err := patch.ValidateVersionName(patches.CurrentVersion); err != nil {
			log.Println(err)
			serv.SetPatchesSummary(patches)
			serv.SetState(server.Normal)
			app.SetNormalState()
			return
		}

		log.Printf("Patch version \"%s\" is available\n", patches.CurrentVersion)

		if app.rejectedPatches.IsRejected(serv, patches.CurrentVersion) {
			log.Printf("Patch version \"%s\" is rejected; Aborting update sequence.\n", patches.CurrentVersion)
			serv.SetState(server.Normal)
			app.SetNormalState()
			return
		}

		serv.SetPatchesSummary(patches)
		serv.SetState(server.PendingUpdate)
		app.SetUpdateState()
	}(serv)
}

func (app *App) IsReady() bool {
	return app.playButton != nil
}

func (app *App) CheckClient() {
	log.Printf("Using \"%s\" as client directory\n", app.settings.Client.Directory)
	app.clientPathBinding.Set(app.settings.ClientPath())

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

func (app *App) CheckPrerequisites() {
	if app.client.MeetsPrerequisites() {
		app.settings.MeetsPrerequisites = true
		app.settings.Save()
		return
	}

	window := prerequisites.New(app, func(b bool) {
		app.settings.MeetsPrerequisites = b
		app.settings.Save()
	})

	window.CenterOnScreen()
	window.RequestFocus()
	window.Show()
}

func (app *App) Start() {
	app.CheckClient()

	if app.settings.CheckPatchesAutomatically {
		app.CheckForUpdates(app.CurrentServer())
	}

	if !app.settings.MeetsPrerequisites {
		go app.CheckPrerequisites()
	}

	app.main.CenterOnScreen()
	app.main.ShowAndRun()
}
