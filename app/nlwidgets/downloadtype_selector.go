package nlwidgets

import (
	"github.com/I-Am-Dench/nimbus-launcher/patcher"
)

func NewDownloadTypeSelector(initial patcher.DownloadType, optional bool) *ItemSelector[patcher.DownloadType] {
	options := []patcher.DownloadType{
		patcher.DownloadTypeMinimal,
		patcher.DownloadTypeFull,
	}

	selector := NewItemSelector(
		options,
		func(d patcher.DownloadType) string {
			switch d {
			case patcher.DownloadTypeMinimal:
				return "Minimal Download (During Play)"
			case patcher.DownloadTypeFull:
				return "Full Download"
			default:
				return d.String()
			}
		},
		func(a, b patcher.DownloadType) bool { return a == b },
		func(patcher.DownloadType) {},
	)
	selector.PlaceHolder = "(Default)"

	if initial >= 0 {
		selector.SetSelected(initial)
	} else if !optional {
		selector.SetSelectedIndex(0)
	}

	return selector
}
