package nlwidgets

import "github.com/I-Am-Dench/nimbus-launcher/locale"

func NewLocaleSelector(initial string, optional bool) *ItemSelector[string] {
	options := locale.List()

	selector := NewItemSelector(
		options,
		func(l string) string {
			return locale.GetName(l)
		},
		func(a, b string) bool { return a == b },
		func(string) {},
	)
	selector.PlaceHolder = "(Default)"

	if len(initial) > 0 {
		selector.SetSelected(initial)
	} else if !optional {
		selector.SetSelectedIndex(0)
	}

	return selector
}
