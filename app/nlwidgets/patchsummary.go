package nlwidgets

import (
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/I-Am-Dench/nimbus-launcher/patcher"
)

func AskContinuePatch(summary patcher.Summary) (ok bool) {
	wg := sync.WaitGroup{}
	wg.Add(1)

	fyne.Do(func() {
		window := fyne.CurrentApp().NewWindow("Patch")
		window.SetIcon(theme.ListIcon())
		window.SetOnClosed(wg.Done)
		window.Resize(fyne.NewSize(600, 500))
		window.SetFixedSize(true)

		heading := canvas.NewText("Patcher will install the following file(s):", theme.Color(theme.ColorNameForeground))
		heading.TextSize = 14

		submit := widget.NewButtonWithIcon("Contine", theme.DownloadIcon(), func() {
			ok = true
			window.Close()
		})
		submit.Importance = widget.HighImportance

		cancel := widget.NewButtonWithIcon("Cancel", theme.CancelIcon(), window.Close)

		buttons := container.NewBorder(nil, nil, nil, container.NewHBox(cancel, submit))

		firstDraw := true
		lastColumnMaxLength := float32(0)
		table := widget.NewTable(
			func() (rows int, cols int) {
				return len(summary.Rows), len(summary.Header)
			},
			func() fyne.CanvasObject {
				l := widget.NewLabel("")
				l.Truncation = fyne.TextTruncateEllipsis
				return l
			},
			func(id widget.TableCellID, template fyne.CanvasObject) {
				l := template.(*widget.Label)
				l.SetText(summary.Rows[id.Row][id.Col])
				if id.Col == len(summary.Header)-1 {
					l.Truncation = fyne.TextTruncateOff
					l.Refresh()
					lastColumnMaxLength = max(lastColumnMaxLength, l.MinSize().Width)
				}
			},
		)
		table.UpdateHeader = func(id widget.TableCellID, template fyne.CanvasObject) {
			template.(*widget.Label).SetText(summary.Header[id.Col])
			if firstDraw { // Fyne tables are annoying. We need to set the sizes by hand, but only on the first update.
				if id.Col < len(summary.Header)-1 {
					table.SetColumnWidth(id.Col, 128)
				} else {
					firstDraw = false
					table.SetColumnWidth(id.Col, lastColumnMaxLength)
				}
			}
		}
		table.ShowHeaderRow = true

		window.SetContent(
			container.NewPadded(
				container.NewBorder(
					heading, buttons, nil, nil, table,
				),
			),
		)

		window.CenterOnScreen()
		window.Show()
	})

	wg.Wait()

	return ok
}
