package nlwidgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

type ItemRadioGroup[T any] struct {
	widget.RadioGroup

	options []T
	strFunc func(T) string

	Selected T
}

func NewItemRadioGroup[T any](options []T, strFunc func(T) string, changed func(T)) *ItemRadioGroup[T] {
	g := &ItemRadioGroup[T]{
		strFunc: strFunc,
	}
	g.ExtendBaseWidget(g)

	g.OnChanged = func(s string) {
		var selected T
		for _, v := range g.options {
			if g.strFunc(v) == s {
				selected = v
				break
			}
		}
		g.Selected = selected
		changed(g.Selected)
	}

	g.SetOptions(options)

	return g
}

func (g *ItemRadioGroup[T]) SetOptions(options []T) {
	g.options = options

	strOptions := []string{}
	for _, v := range options {
		strOptions = append(strOptions, g.strFunc(v))
	}

	g.RadioGroup.Options = strOptions
	fyne.Do(func() {
		g.RadioGroup.Refresh()
		if len(g.RadioGroup.Selected) > 0 {
			g.OnChanged(g.RadioGroup.Selected)
		}
	})
}
