package nlwidgets

import (
	"github.com/I-Am-Dench/nimbus-launcher/locale"
)

func NewLocaleSelector(initial string, optional ...bool) *ItemSelector[string] {
	options := locale.List()
	if len(optional) > 0 && optional[0] {
		options = append([]string{""}, options...)
	}

	selector := NewItemSelector(
		options,
		func(l string) string {
			if len(l) == 0 {
				return "(Default)"
			}
			return locale.GetName(l)
		},
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
