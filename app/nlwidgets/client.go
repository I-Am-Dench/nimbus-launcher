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

	return client.Optional{
		Directory:    c.installDirectory.Text,
		Name:         c.clientName.Text,
		IsPacked:     isPacked,
		Locale:       c.locale.Selected,
		DownloadType: downloadType,
	}
}

func (c ClientSettings) Get() client.Config {
	return client.Config{
		Directory:    c.installDirectory.Text,
		Name:         c.clientName.Text,
		IsPacked:     !c.packed.Partial && c.packed.Checked,
		Locale:       c.locale.Selected,
		DownloadType: c.downloadType.Selected,
	}
}
