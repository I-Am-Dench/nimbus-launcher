package app

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func (app *App) LoadContent() {
	heading := canvas.NewText("Launch LEGO Universe", theme.ForegroundColor())
	heading.TextSize = 24

	infoButton := widget.NewButtonWithIcon(
		"", theme.InfoIcon(), app.ShowInfo,
	)
	infoButton.Importance = widget.LowImportance

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
				container.NewHBox(heading, infoButton),
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
