package nlwidgets

import (
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/I-Am-Dench/nimbus-launcher/client"
)

func NewDirectorySelector(window fyne.Window) *widget.Entry {
	entry := widget.NewEntry()
	entry.PlaceHolder = client.DefaultDir

	button := widget.NewButtonWithIcon(
		"", theme.FolderOpenIcon(), func() {
			dialog.ShowFolderOpen(func(lu fyne.ListableURI, err error) {
				if err != nil {
					dialog.ShowError(err, window)
					return
				}

				if lu == nil {
					return
				}

				entry.SetText(filepath.Clean(lu.Path()))
			}, window)
		},
	)
	button.Importance = widget.LowImportance
	entry.ActionItem = button

	return entry
}
