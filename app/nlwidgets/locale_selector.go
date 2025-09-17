package nlwidgets

import (
	"github.com/I-Am-Dench/nimbus-launcher/locale"
)

func NewLocaleSelector(initial string) *ItemSelector[string] {
	selector := NewItemSelector(
		locale.List(),
		func(l string) string { return locale.GetName(l) },
		func(a, b string) bool { return a == b },
		func(_ string) {},
	)

	if len(initial) > 0 {
		selector.SetSelected(initial)
	} else {
		selector.SetSelectedIndex(0)
	}

	return selector
}
