package nlwidgets

import (
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type BinaryProgressBar struct {
	definiteProgress   *widget.ProgressBar
	indefiniteProgress *widget.ProgressBarInfinite
	textFormat         string
}

func NewBinaryProgressBar() *BinaryProgressBar {
	progress := new(BinaryProgressBar)

	progress.definiteProgress = widget.NewProgressBar()
	progress.definiteProgress.TextFormatter = progress.GetFormat

	progress.indefiniteProgress = widget.NewProgressBarInfinite()

	return progress
}

// The text formatted returned by ProgressBar.TextFormatter
//
// $VALUE gets replaced with int(ProgressBar.Value)
//
// $MAX gets replaced with int(ProgressBar.Max)
func (progress *BinaryProgressBar) SetFormat(format string) {
	progress.textFormat = format
}

func (progress *BinaryProgressBar) GetFormat() string {
	withValue := strings.ReplaceAll(progress.textFormat, "$VALUE", strconv.Itoa(int(progress.Value())))
	withMax := strings.ReplaceAll(withValue, "$MAX", strconv.Itoa(int(progress.definiteProgress.Max)))
	return withMax
}

func (progress *BinaryProgressBar) ShowDefinite() {
	progress.definiteProgress.Show()
	progress.indefiniteProgress.Hide()
}

func (progress *BinaryProgressBar) ShowValue(value float64, format string) {
	progress.definiteProgress.SetValue(value)
	progress.textFormat = format
	progress.ShowDefinite()
}

func (progress *BinaryProgressBar) ShowFormat(format string) {
	progress.textFormat = format
	progress.ShowDefinite()
}

func (progress *BinaryProgressBar) ShowIndefinite() {
	progress.indefiniteProgress.Show()
	progress.definiteProgress.Hide()
}

func (progress *BinaryProgressBar) SetMax(max float64) {
	progress.definiteProgress.Max = max
}

func (progress *BinaryProgressBar) SetValue(value float64) {
	progress.definiteProgress.SetValue(value)
}

func (progress *BinaryProgressBar) Value() float64 {
	return progress.definiteProgress.Value
}

func (progress *BinaryProgressBar) Container() *fyne.Container {
	return container.NewStack(progress.definiteProgress, progress.indefiniteProgress)
}

func (progress *BinaryProgressBar) Hide() {
	progress.definiteProgress.Hide()
	progress.indefiniteProgress.Hide()
}
