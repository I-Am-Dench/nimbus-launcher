package nldialogs

import (
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/I-Am-Dench/nimbus-launcher/app/nlwidgets"
)

func AskContinueOnError(err error) (ok bool) {
	wg := sync.WaitGroup{}
	wg.Add(1)

	fyne.Do(func() {
		window := fyne.CurrentApp().NewWindow("Patcher Error")
		window.SetIcon(theme.ErrorIcon())
		window.SetOnClosed(func() {
			wg.Done()
		})
		window.Resize(fyne.NewSize(500, 500))
		window.SetFixedSize(true)

		heading := canvas.NewText("Patcher returned the following error(s):", theme.Color(theme.ColorNameForeground))
		heading.TextSize = 14

		submit := widget.NewButtonWithIcon("Launch Anyway", theme.MediaPlayIcon(), func() {
			ok = true
			window.Close()
		})
		submit.Importance = widget.HighImportance

		cancel := widget.NewButtonWithIcon("Stop", theme.MediaStopIcon(), func() {
			window.Close()
		})

		buttons := container.NewBorder(nil, nil, nil, container.NewHBox(cancel, submit))

		errorMessage := nlwidgets.NewCodeBox()
		errorMessage.SetText(err.Error())

		window.SetContent(
			container.NewPadded(
				container.NewBorder(
					heading, buttons, nil, nil, container.NewStack(
						errorMessage,
					),
				),
			),
		)

		window.CenterOnScreen()
		window.Show()
	})

	wg.Wait()

	return ok
}
