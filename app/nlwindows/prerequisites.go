package nlwindows

import (
	"fmt"
	"runtime"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const linuxPrerequisites = `This launcher utilizes [https://github.com/ValveSoftware/Proton](https://github.com/ValveSoftware/Proton) for launching LEGO Universe clients,

but a viable Proton installation could not be found on your system.

Proton is a tool created by Steam for launching Windows compatible games on Linux, and may be

installed via the **Compatibility** tab within Steam settings or by sorting by **Tools** in the Library tab.`

func prereqContainer() *fyne.Container {
	switch os := runtime.GOOS; os {
	case "linux":
		return container.NewStack(
			widget.NewRichTextFromMarkdown(linuxPrerequisites),
		)
	default:
		return container.NewStack(
			widget.NewLabel(fmt.Sprintf("Unsupported OS: %s", os)),
		)
	}
}

func NewPrerequisitesWindow(app fyne.App, onClose func(bool)) fyne.Window {
	window := app.NewWindow("Missing Prerequisites")
	window.SetIcon(theme.WarningIcon())
	window.SetFixedSize(true)

	heading := canvas.NewText("Missing Prerequisites", theme.ForegroundColor())
	heading.TextSize = 16

	dontShowAgain := widget.NewCheck("Don't Show Again", func(b bool) {})
	ok := widget.NewButton("Done", func() {
		onClose(dontShowAgain.Checked)
		window.Close()
	})

	window.SetContent(
		container.NewPadded(
			container.NewVBox(
				heading, widget.NewSeparator(),
				container.NewHBox(
					prereqContainer(),
				),
				container.NewBorder(nil, nil, dontShowAgain, ok),
			),
		),
	)

	return window
}
