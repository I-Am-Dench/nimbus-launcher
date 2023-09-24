package app

import (
	"image/color"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func (app *App) LoadContent() {
	heading := canvas.NewText("Launch Lego Universe", color.White)
	heading.TextSize = 24

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
			container.NewBorder(nil, nil, nil, app.refreshUpdatesButton),
		),
	)

	innerContent := container.NewPadded(
		container.NewVBox(
			container.NewBorder(
				nil, nil, nil,
				addServerButton,
				app.serverList,
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
	clientLabel := widget.NewLabelWithData(app.clientPathBinding)
	clientLabel.Alignment = fyne.TextAlignLeading
	clientLabel.TextStyle = fyne.TextStyle{
		Bold: true,
	}
	clientLabel.Truncation = fyne.TextTruncateEllipsis

	return container.NewBorder(
		app.progressBar.Container(),
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
	return container.NewPadded(
		NewServersPage(window, app.serverList).Container(),
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

	reviewPatchBeforeUpdate := widget.NewCheck("", func(b bool) {})
	reviewPatchBeforeUpdate.Checked = app.settings.ReviewPatchBeforeUpdate

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

	// runCommand := widget.NewEntry()
	// runCommand.PlaceHolder = "wine or wine64"

	// environmentVariables := widget.NewEntry()
	// environmentVariables.PlaceHolder = "Separated by ;"

	saveButton := widget.NewButton("Save", func() {
		app.settings.CloseOnPlay = closeOnPlay.Checked
		app.settings.CheckPatchesAutomatically = checkPatchesAutomatically.Checked
		app.settings.ReviewPatchBeforeUpdate = reviewPatchBeforeUpdate.Checked

		app.settings.Client.Directory = clientDirectory.Text
		app.settings.Client.Name = clientName.Text
		// app.settings.Client.RunCommand = runCommand.Text
		// app.settings.Client.EnvironmentVariables = environmentVariables.Text

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
						widget.NewFormItem("Review Patch Before Update", reviewPatchBeforeUpdate),
					),
					widget.NewSeparator(),
					clientHeading,
					widget.NewForm(
						widget.NewFormItem("Directory", clientDirectory),
						widget.NewFormItem("Name", clientName),
						// widget.NewFormItem("Run Command", runCommand),
						// widget.NewFormItem("EnvironmentVariables", environmentVariables),
					),
				),
			),
		),
	)
}
