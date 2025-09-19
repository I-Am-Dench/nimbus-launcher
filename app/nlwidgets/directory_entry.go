package nlwidgets

import (
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/I-Am-Dench/nimbus-launcher/client"
)

func NewDirectorySelector(window fyne.Window, onSelect ...func(string)) *widget.Entry {
	var f func(string)
	if len(onSelect) > 0 {
		f = onSelect[0]
	}

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

				path := filepath.Clean(lu.Path())
				entry.SetText(path)
				if f != nil {
					f(path)
				}
			}, window)
		},
	)
	button.Importance = widget.LowImportance
	entry.ActionItem = button

	return entry
}
