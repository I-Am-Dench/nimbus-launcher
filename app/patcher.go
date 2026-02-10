package app

import (
	"slices"

	"fyne.io/fyne/v2/widget"
	"github.com/I-Am-Dench/nimbus-launcher/patcher"
	"github.com/I-Am-Dench/nimbus-launcher/patcher/protocols/netdevil"
	"github.com/I-Am-Dench/nimbus-launcher/patcher/protocols/nimbus"
)

type PatcherFunc = func() Patcher

type Patcher interface {
	patcher.Environment

	Default() Patcher
	ProfileForm() ([]*widget.FormItem, PatcherFunc)
	UserForm() ([]*widget.FormItem, PatcherFunc)
}

var Patchers = map[string]Patcher{
	"netdevil": NetDevilPatcher{},
	"nimbus":   NimbusPatcher{},
}

func PatcherOptions() []string {
	options := []string{}
	for id := range Patchers {
		options = append(options, id)
	}
	slices.Sort(options)

	return append([]string{"(None)"}, options...)
}

type NetDevilPatcher struct {
	netdevil.Environment
}

func (e NetDevilPatcher) Default() Patcher {
	return &NetDevilPatcher{
		Environment: netdevil.Environment{
			Environment: "live",
		},
	}
}

func (e NetDevilPatcher) ProfileForm() ([]*widget.FormItem, PatcherFunc) {
	environment := widget.NewEntry()
	environment.SetText(e.Environment.Environment)

	return []*widget.FormItem{
			widget.NewFormItem("Environment", environment),
		}, func() Patcher {
			return &NetDevilPatcher{
				netdevil.Environment{
					Environment: environment.Text,
				},
			}
		}
}

func (e NetDevilPatcher) UserForm() ([]*widget.FormItem, PatcherFunc) {
	return []*widget.FormItem{}, func() Patcher {
		return &NetDevilPatcher{
			netdevil.Environment{
				Environment: e.Environment.Environment,
			},
		}
	}
}

type NimbusPatcher struct {
	nimbus.Environment
}

func (e NimbusPatcher) Default() Patcher {
	return &NimbusPatcher{
		Environment: nimbus.Environment{
			Environment: "live",
		},
	}
}

func (e NimbusPatcher) ProfileForm() ([]*widget.FormItem, PatcherFunc) {
	environment := widget.NewEntry()
	environment.SetText(e.Environment.Environment)

	return []*widget.FormItem{
			widget.NewFormItem("Environment", environment),
		}, func() Patcher {
			return &NimbusPatcher{
				nimbus.Environment{
					Environment: environment.Text,
				},
			}
		}
}

func (e NimbusPatcher) UserForm() ([]*widget.FormItem, PatcherFunc) {
	return []*widget.FormItem{}, func() Patcher {
		return &NimbusPatcher{e.Environment}
	}
}
