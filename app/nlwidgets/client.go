package nlwidgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/I-Am-Dench/nimbus-launcher/client"
	"github.com/I-Am-Dench/nimbus-launcher/internal/optional"
)

type ClientSettings struct {
	installDirectory *widget.Entry
	clientName       *widget.Entry
	packed           *widget.Check
	locale           *ItemSelector[string]
	fullDownload     *widget.Check

	resetPacked       *widget.Button
	resetFullDownload *widget.Button

	isDefault bool
}

func NewClientSettings(window fyne.Window, isDefault bool, config ...client.Optional) ClientSettings {
	c := ClientSettings{
		installDirectory: NewDirectorySelector(window),
		clientName:       widget.NewEntry(),
		packed:           widget.NewCheck("Uses catalog (i.e. versions/primary.pki)", func(b bool) {}),
		locale:           NewLocaleSelector("", !isDefault),
		fullDownload:     widget.NewCheck("Download before or during play", func(b bool) {}),

		isDefault: isDefault,
	}
	c.clientName.PlaceHolder = "client/legouniverse.exe"

	c.resetPacked = widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
		c.packed.Partial = true
		c.packed.Refresh()
	})
	c.resetPacked.Hidden = isDefault

	c.resetFullDownload = widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
		c.fullDownload.Partial = true
		c.fullDownload.Refresh()
	})
	c.resetFullDownload.Hidden = isDefault

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
		widget.NewFormItem("Locale", c.locale),
		widget.NewFormItem("Full Download", container.NewBorder(nil, nil, c.fullDownload, c.resetFullDownload)),
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

	c.fullDownload.Partial = !c.isDefault && !config.FullDownload.HasValue()
	if !c.fullDownload.Partial {
		c.fullDownload.SetChecked(config.FullDownload.Value)
	}
}

func (c ClientSettings) GetOptional() client.Optional {
	isPacked := optional.O[bool]{}
	if !c.packed.Partial {
		isPacked = optional.From(c.packed.Checked)
	}

	isFullDownload := optional.O[bool]{}
	if !c.fullDownload.Partial {
		isFullDownload = optional.From(c.fullDownload.Checked)
	}

	return client.Optional{
		Directory:    c.installDirectory.Text,
		Name:         c.clientName.Text,
		IsPacked:     isPacked,
		Locale:       c.locale.Selected,
		FullDownload: isFullDownload,
	}
}

func (c ClientSettings) Get() client.Config {
	return client.Config{
		Directory:    c.installDirectory.Text,
		Name:         c.clientName.Text,
		IsPacked:     !c.packed.Partial && c.packed.Checked,
		Locale:       c.locale.Selected,
		FullDownload: !c.fullDownload.Partial && c.fullDownload.Checked,
	}
}
