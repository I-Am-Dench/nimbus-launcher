package nlwidgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
)

type ProgressBar struct {
	Container *fyne.Container

	progress *widget.ProgressBar
	infinite *widget.ProgressBarInfinite

	infiniteBinding binding.String
	infiniteLabel   *widget.Label
}

func NewProgressBar() *ProgressBar {
	progress := widget.NewProgressBar()
	infinite := widget.NewProgressBarInfinite()

	infiniteBinding := binding.NewString()
	infiniteLabel := widget.NewLabelWithData(infiniteBinding)
	infiniteLabel.Truncation = fyne.TextTruncateEllipsis

	progress.Hide()
	infinite.Hide()

	progress.TextFormatter = func() string {
		text, _ := infiniteBinding.Get()
		return text
	}

	return &ProgressBar{
		Container: container.NewStack(progress, infinite, infiniteLabel),

		progress:        progress,
		infinite:        infinite,
		infiniteBinding: infiniteBinding,
		infiniteLabel:   infiniteLabel,
	}
}

func (p *ProgressBar) SetMax(f float64) {
	p.progress.Max = f
}

func (p *ProgressBar) SetValue(f float64) {
	p.progress.SetValue(f)
}

func (p *ProgressBar) SetText(s string) {
	p.infiniteBinding.Set(s)
}

func (p *ProgressBar) HideProgress() {
	p.progress.Hide()
	p.infinite.Hide()
	p.infiniteLabel.Hide()
}

func (p *ProgressBar) Infinite() {
	p.progress.Hide()
	p.infinite.Show()
	p.infiniteLabel.Show()
}

func (p *ProgressBar) Progress() {
	p.progress.Show()
	p.infinite.Hide()
	p.infiniteLabel.Hide()
}
