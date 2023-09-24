package app

import (
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func (app *App) SetPlayingState() {
	app.progressBar.Hide()

	app.playButton.Disable()
	app.playButton.SetText("Playing")

	app.serverList.Disable()
}

func (app *App) SetNormalState() {
	app.progressBar.Hide()

	app.playButton.Enable()
	app.playButton.SetText("Play")
	app.playButton.SetIcon(theme.MediaPlayIcon())
	app.playButton.Importance = widget.HighImportance
	app.playButton.OnTapped = app.PressPlay
	app.playButton.Refresh()

	app.refreshUpdatesButton.Enable()

	app.serverList.Enable()
}

func (app *App) SetUpdatingState() {
	app.progressBar.ShowIndefinite()

	app.playButton.Disable()
	app.playButton.SetText("Updating")

	app.refreshUpdatesButton.Disable()
}

func (app *App) SetUpdateState() {
	app.progressBar.Hide()

	app.playButton.Enable()
	app.playButton.SetText("Update")
	app.playButton.SetIcon(theme.DownloadIcon())
	app.playButton.Importance = widget.SuccessImportance
	app.playButton.OnTapped = app.PressUpdate
	app.playButton.Refresh()

	app.refreshUpdatesButton.Enable()
}

func (app *App) SetCheckingUpdatesState() {
	app.progressBar.ShowIndefinite()

	app.playButton.Disable()
	app.playButton.SetText("Checking updates")
	app.playButton.SetIcon(nil)
	app.playButton.Refresh()

	app.refreshUpdatesButton.Disable()
}
