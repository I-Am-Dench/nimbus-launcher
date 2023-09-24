package luwindows

import (
	"encoding/json"
	"fmt"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/I-Am-Dench/lu-launcher/luwidgets"
	"github.com/I-Am-Dench/lu-launcher/resource"
)

type PatchAcceptState uint32

const (
	PatchAccept = PatchAcceptState(iota)
	PatchCancel
	PatchReject
)

func NewPatchReviewWindow(app fyne.App, patch resource.Patch, onConfirmCancel func(PatchAcceptState)) fyne.Window {
	window := app.NewWindow("Review Patch")
	window.SetFixedSize(true)
	window.Resize(fyne.NewSize(800, 600))
	window.SetIcon(theme.QuestionIcon())

	LoadPatchReviewContainer(window, patch, onConfirmCancel)

	return window
}

func LoadPatchReviewContainer(window fyne.Window, patch resource.Patch, onConfirmCancel func(PatchAcceptState)) {
	heading := canvas.NewText(fmt.Sprintf("Received patch.json (%s):", patch.Version), color.White)
	heading.TextSize = 16

	reject := widget.NewButton(
		"Reject", func() {
			window.Close()
			onConfirmCancel(PatchReject)
		},
	)
	reject.Importance = widget.DangerImportance

	confirm := widget.NewButton(
		"Continue", func() {
			window.Close()
			onConfirmCancel(PatchAccept)
		},
	)
	confirm.Importance = widget.HighImportance

	cancel := widget.NewButton(
		"Cancel", func() {
			window.Close()
			onConfirmCancel(PatchCancel)
		},
	)

	footer := container.NewBorder(
		widget.NewLabelWithStyle(
			"Continue with update?",
			fyne.TextAlignLeading,
			fyne.TextStyle{
				Bold: true,
			},
		), nil,
		reject, container.NewHBox(cancel, confirm),
	)

	data, _ := json.MarshalIndent(patch, "", "    ")
	patchContent := luwidgets.NewCodeBox()
	patchContent.SetText(string(data))

	window.SetContent(
		container.NewPadded(
			container.NewBorder(
				heading, footer,
				nil, nil,
				container.NewVScroll(
					patchContent,
				),
			),
		),
	)
}
