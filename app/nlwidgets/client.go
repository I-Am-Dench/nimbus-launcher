package nlwidgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/I-Am-Dench/nimbus-launcher/client"
	"github.com/I-Am-Dench/nimbus-launcher/internal/optional"
	"github.com/I-Am-Dench/nimbus-launcher/patcher"
)

type ClientSettings struct {
	installDirectory *widget.Entry
	clientName       *widget.Entry
	packed           *widget.Check
	locale           *ItemSelector[string]
	downloadType     *ItemSelector[patcher.DownloadType]
	maxUgcSize       *IntegerEntry
	infiniteUgcSize  *widget.Check

	resetPacked       *widget.Button
	resetLocale       *widget.Button
	resetDownloadType *widget.Button

	isDefault bool
}

func NewClientSettings(window fyne.Window, isDefault bool, config ...client.Optional) ClientSettings {
	c := ClientSettings{
		installDirectory: NewDirectorySelector(window),
		clientName:       widget.NewEntry(),
		packed:           widget.NewCheck("Uses catalog (i.e. versions/primary.pki)", func(b bool) {}),
		locale:           NewLocaleSelector("", !isDefault),
		downloadType:     NewDownloadTypeSelector(-1, !isDefault),
		maxUgcSize:       NewIntegerEntry(),

		isDefault: isDefault,
	}
	c.clientName.PlaceHolder = "client/legouniverse.exe"

	c.resetPacked = widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
		c.packed.Partial = true
		c.packed.Refresh()
	})
	c.resetPacked.Hidden = isDefault

	c.resetLocale = widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
		c.locale.ClearSelected()
	})
	c.resetLocale.Hidden = isDefault

	c.resetDownloadType = widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
		c.downloadType.ClearSelected()
	})
	c.resetDownloadType.Hidden = isDefault

	c.maxUgcSize.ActionItem = widget.NewLabel("GiB")
	c.infiniteUgcSize = widget.NewCheck("No Max Size", func(b bool) {
		if b {
			c.maxUgcSize.Disable()
		} else {
			c.maxUgcSize.Enable()
		}
	})

	if len(config) > 0 {
		c.Set(config[0])
	}
	return c
}

func (c ClientSettings) Form() []*widget.FormItem {
	return []*widget.FormItem{
		widget.NewFormItem("Directory", c.installDirectory),
		widget.NewFormItem("Name", c.clientName),
		widget.NewFormItem("Packed", container.NewBorder(nil, nil, c.packed, c.resetPacked)),
		widget.NewFormItem("Locale", container.NewBorder(nil, nil, nil, c.resetLocale, c.locale)),
		widget.NewFormItem("Download Type", container.NewBorder(nil, nil, nil, c.resetDownloadType, c.downloadType)),
		widget.NewFormItem("Max UGC Size", container.NewVBox(c.maxUgcSize, c.infiniteUgcSize)),
	}
}

func (c ClientSettings) Set(config client.Optional) {
	c.installDirectory.SetText(config.Directory)
	c.clientName.SetText(config.Name)

	c.packed.Partial = !c.isDefault && !config.IsPacked.HasValue()
	if !c.packed.Partial {
		c.packed.SetChecked(config.IsPacked.Value)
	}

	c.locale.SetSelected(config.Locale)

	if config.DownloadType.HasValue() {
		c.downloadType.SetSelected(config.DownloadType.Value)
	}

	if config.MaxUgcSpace.HasValue() {
		c.maxUgcSize.SetValue(int64(config.MaxUgcSpace.Value))
	}

	if !config.MaxUgcSpace.HasValue() {
		c.infiniteUgcSize.SetChecked(true)
	}
}

func (c ClientSettings) GetOptional() client.Optional {
	isPacked := optional.O[bool]{}
	if !c.packed.Partial {
		isPacked = optional.From(c.packed.Checked)
	}

	downloadType := optional.O[patcher.DownloadType]{}
	if c.downloadType.SelectedIndex() >= 0 {
		downloadType = optional.From(c.downloadType.Selected)
	}

	maxUgcSize := optional.O[int]{}
	if !c.infiniteUgcSize.Checked {
		maxUgcSize = optional.From(int(c.maxUgcSize.Value()))
	}

	return client.Optional{
		Directory:    c.installDirectory.Text,
		Name:         c.clientName.Text,
		IsPacked:     isPacked,
		Locale:       c.locale.Selected,
		DownloadType: downloadType,
		MaxUgcSpace:  maxUgcSize,
	}
}

func (c ClientSettings) Get() client.Config {
	maxUgcSize := 0
	if !c.infiniteUgcSize.Checked {
		maxUgcSize = int(c.maxUgcSize.Value())
	}

	return client.Config{
		Directory:    c.installDirectory.Text,
		Name:         c.clientName.Text,
		IsPacked:     !c.packed.Partial && c.packed.Checked,
		Locale:       c.locale.Selected,
		DownloadType: c.downloadType.Selected,
		MaxUgcSpace:  maxUgcSize,
	}
}
