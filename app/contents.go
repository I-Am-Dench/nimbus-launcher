package app

import (
	"fmt"
	"image/color"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/I-Am-Dench/lu-launcher/luconfig"
	"github.com/I-Am-Dench/lu-launcher/resource"
)

func (app *App) LoadContent() {
	heading := canvas.NewText("Launch Lego Universe", color.White)
	heading.TextSize = 24

	app.serverSelector = widget.NewSelect(
		app.servers.Names(),
		func(s string) {
			app.SetCurrentServer(app.serverSelector.SelectedIndex())
		},
	)
	app.serverSelector.SetSelectedIndex(app.settings.SelectedServer)

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

	accountInfo := container.NewBorder(
		nil, nil,
		container.NewVBox(
			HyperLinkButton("Signup", theme.AccountIcon(), app.signupBinding),
			HyperLinkButton("Signin", theme.LoginIcon(), app.signinBinding),
		),
		nil,
		container.NewVBox(
			AddEllipsis(widget.NewLabelWithData(app.signupBinding)),
			AddEllipsis(widget.NewLabelWithData(app.signinBinding)),
		),
	)

	innerContent := container.NewPadded(
		container.NewVBox(
			container.NewBorder(
				nil, nil, nil,
				addServerButton,
				app.serverSelector,
			),
			container.NewGridWithColumns(
				2, serverInfo, accountInfo,
			),
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
	// if !app.FoundClient {
	// 	app.playButton.Disable()
	// }

	clientLabel := widget.NewLabelWithStyle(
		app.settings.ClientPath(),
		fyne.TextAlignLeading,
		fyne.TextStyle{
			Bold: true,
		},
	)
	clientLabel.Truncation = fyne.TextTruncateEllipsis

	app.definiteProgress = widget.NewProgressBar()
	app.definiteProgress.TextFormatter = func() string {
		return app.progressText
	}
	app.definiteProgress.Hide()

	app.indefiniteProgress = widget.NewProgressBarInfinite()
	app.indefiniteProgress.Hide()

	prepareProgressBar := container.NewStack(
		app.definiteProgress,
		app.indefiniteProgress,
	)

	return container.NewBorder(
		prepareProgressBar,
		nil,
		app.clientErrorIcon,
		app.playButton,
		clientLabel,
	)
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

	clientHeading := canvas.NewText("Client", color.White)
	clientHeading.TextSize = 16

	closeOnPlay := widget.NewCheck("", func(b bool) {})
	closeOnPlay.Checked = app.settings.CloseOnPlay

	checkPatchesAutomatically := widget.NewCheck("", func(b bool) {})
	checkPatchesAutomatically.Checked = app.settings.CheckPatchesAutomatically

	clientDirectory := widget.NewEntry()
	clientDirectoryButton := widget.NewButtonWithIcon(
		"", theme.FolderOpenIcon(), func() {
			dialog.ShowFolderOpen(func(lu fyne.ListableURI, err error) {
				if err != nil {
					dialog.ShowError(err, window)
					return
				}

				if lu == nil {
					return
				}

				clientDirectory.SetText(filepath.Clean(lu.Path()))
			}, window)
		},
	)
	clientDirectoryButton.Importance = widget.LowImportance
	clientDirectory.ActionItem = clientDirectoryButton

	clientDirectory.SetText(app.settings.Client.Directory)

	clientName := widget.NewEntry()
	clientName.PlaceHolder = ".exe"
	clientName.SetText(app.settings.Client.Name)

	saveButton := widget.NewButton("Save", func() {
		app.settings.CloseOnPlay = closeOnPlay.Checked
		app.settings.CheckPatchesAutomatically = checkPatchesAutomatically.Checked

		app.settings.Client.Directory = clientDirectory.Text
		app.settings.Client.Name = clientName.Text

		err := app.settings.Save()
		if err != nil {
			dialog.ShowError(err, window)
		} else {
			dialog.ShowInformation("Launcher Settings", "Settings saved!", window)
			app.CheckClient()
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
						widget.NewFormItem("Check Patches Automatically", checkPatchesAutomatically),
					),
					widget.NewSeparator(),
					clientHeading,
					widget.NewForm(
						widget.NewFormItem("Directory", clientDirectory),
						widget.NewFormItem("Name", clientName),
					),
				),
			),
		),
	)
}
