package settings

import (
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/I-Am-Dench/nimbus-launcher/resource"
)

func NewLauncherTab(window fyne.Window, settings *resource.Settings, onSave func()) *container.TabItem {
	generalHeading := canvas.NewText("General", theme.ForegroundColor())
	generalHeading.TextSize = 16

	clientHeading := canvas.NewText("Client", theme.ForegroundColor())
	clientHeading.TextSize = 16

	closeOnPlay := widget.NewCheck("", func(b bool) {})
	closeOnPlay.Checked = settings.CloseOnPlay

	checkPatchesAutomatically := widget.NewCheck("", func(b bool) {})
	checkPatchesAutomatically.Checked = settings.CheckPatchesAutomatically

	reviewPatchBeforeUpdate := widget.NewCheck("", func(b bool) {})
	reviewPatchBeforeUpdate.Checked = settings.ReviewPatchBeforeUpdate

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

	clientDirectory.SetText(settings.Client.Directory)

	clientName := widget.NewEntry()
	clientName.PlaceHolder = ".exe"
	clientName.SetText(settings.Client.Name)

	saveButton := widget.NewButton("Save", func() {
		settings.CloseOnPlay = closeOnPlay.Checked
		settings.CheckPatchesAutomatically = checkPatchesAutomatically.Checked
		settings.ReviewPatchBeforeUpdate = reviewPatchBeforeUpdate.Checked

		settings.Client.Directory = clientDirectory.Text
		settings.Client.Name = clientName.Text

		if err := settings.Save(); err != nil {
			dialog.ShowError(err, window)
		} else {
			dialog.ShowInformation("Launcher Settings", "Settings saved!", window)
			onSave()
		}
	})
	saveButton.Importance = widget.HighImportance

	content := container.NewPadded(
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
					),
				),
			),
		),
	)

	return container.NewTabItem(
		"Launcher", container.NewPadded(content),
	)
}
