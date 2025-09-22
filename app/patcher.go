package app

import (
	"slices"

	"fyne.io/fyne/v2/widget"
	"github.com/I-Am-Dench/nimbus-launcher/app/nlwidgets"
	"github.com/I-Am-Dench/nimbus-launcher/patcher"
	"github.com/I-Am-Dench/nimbus-launcher/patcher/protocols/netdevil"
)

type PatcherFunc = func() Patcher

type Patcher interface {
	patcher.Environment

	Default() Patcher
	ProfileForm() ([]*widget.FormItem, PatcherFunc)
	UserForm() ([]*widget.FormItem, PatcherFunc)
}

var Patchers = map[string]Patcher{
	"nd-nimbus": &NdNimbusPatcher{},
}

func PatcherOptions() []string {
	options := []string{}
	for id := range Patchers {
		options = append(options, id)
	}
	slices.Sort(options)

	return append([]string{"(None)"}, options...)
}

type NdNimbusPatcher struct {
	netdevil.Environment
}

func (e *NdNimbusPatcher) Default() Patcher {
	return &NdNimbusPatcher{
		Environment: netdevil.Environment{
			Environment: "live",
			UserConfig: netdevil.UserConfig{
				Locale:       "en_US",
				FullDownload: true,
			},
		},
	}
}

func (e *NdNimbusPatcher) ProfileForm() ([]*widget.FormItem, PatcherFunc) {
	environment := widget.NewEntry()
	environment.SetText(e.Environment.Environment)

	return []*widget.FormItem{
			widget.NewFormItem("Environment", environment),
		}, func() Patcher {
			return &NdNimbusPatcher{
				netdevil.Environment{
					Environment: environment.Text,
					UserConfig:  e.UserConfig,
				},
			}
		}
}

func (e *NdNimbusPatcher) UserForm() ([]*widget.FormItem, PatcherFunc) {
	locale := nlwidgets.NewLocaleSelector(e.Locale())

	fullDownload := widget.NewCheck("Download before or during play", func(b bool) {})
	fullDownload.SetChecked(e.FullDownload)

	return []*widget.FormItem{
			widget.NewFormItem("Locale", locale),
			widget.NewFormItem("Full Download", fullDownload),
		}, func() Patcher {
			return &NdNimbusPatcher{
				netdevil.Environment{
					Environment: e.Environment.Environment,
					UserConfig: netdevil.UserConfig{
						Locale:       locale.Selected,
						FullDownload: fullDownload.Checked,
					},
				},
			}
		}
}
