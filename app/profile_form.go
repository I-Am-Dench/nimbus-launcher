package app

import (
	"encoding/xml"
	"fmt"
	"os"
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/I-Am-Dench/goverbuild/encoding/ldf"
	"github.com/I-Am-Dench/goverbuild/models/boot"
	"github.com/I-Am-Dench/nimbus-launcher/client"
)

type ServerXml struct {
	XMLName xml.Name       `xml:"server"`
	Name    string         `xml:"name"`
	Patcher *PatcherConfig `xml:"patcher,omitempty"`
	Boot    struct {
		Text []byte `xml:",innerxml"`
	} `xml:"boot"`
}

type ProfileForm struct {
	Container *fyne.Container

	id          string
	name        *widget.Entry
	patcherType *widget.Select
	serviceUrl  *widget.Entry
	patcherFunc func() Patcher
	bootForm    *BootForm

	client *client.Optional

	patcherContainer *fyne.Container
	patcherForm      *widget.Form
}

func (p *ProfileForm) Set(profile Profile) {
	p.id = profile.Id
	p.name.SetText(profile.Name)

	p.client = profile.Client

	if profile.Server.Patcher != nil {
		p.patcherType.SetSelected(profile.Server.Patcher.Id)
		p.serviceUrl.SetText(profile.Server.Patcher.ServiceUrl)
	}
}

func (p ProfileForm) Get() (*Profile, *boot.Config) {
	id := p.id
	if len(id) == 0 {
		id = strconv.FormatInt(time.Now().Unix(), 10)
	}

	var patcherConfig *PatcherConfig
	if p.patcherType.Selected != "(None)" {
		patcherConfig = &PatcherConfig{
			Id:         p.patcherType.Selected,
			ServiceUrl: p.serviceUrl.Text,
		}

		if p.patcherFunc != nil {
			patcherConfig.Environment = p.patcherFunc()
		}
	}

	return &Profile{
		Id:     id,
		Name:   p.name.Text,
		Client: p.client,
		Server: ServerInfo{
			Patcher: patcherConfig,
		},
	}, p.bootForm.Get()
}

func (p ProfileForm) GetXml() ServerXml {
	profile, bootConfig := p.Get()

	bootData, _ := ldf.MarshalText(bootConfig)

	serverXml := ServerXml{
		Name:    profile.Name,
		Patcher: profile.Server.Patcher,
	}
	serverXml.Boot.Text = bootData

	return serverXml
}

func (p *ProfileForm) SetXml(serverXml ServerXml) error {
	p.name.SetText(serverXml.Name)

	bootConfig := boot.Config{}
	if err := ldf.UnmarshalText(serverXml.Boot.Text, &bootConfig); err != nil {
		return err
	}
	p.bootForm.Set(bootConfig)

	if serverXml.Patcher == nil {
		return nil
	}

	_, ok := Patchers[serverXml.Patcher.Id]
	if !ok {
		return nil
	}

	p.serviceUrl.SetText(serverXml.Patcher.ServiceUrl)
	p.patcherType.SetSelected(serverXml.Patcher.Id)

	p.SetPatcherForm(serverXml.Patcher)

	return nil
}

func (p *ProfileForm) SetPatcherForm(config *PatcherConfig) {
	p.patcherFunc = nil

	if config == nil {
		p.serviceUrl.SetText("")
		p.patcherContainer.Hide()
		return
	}

	patcher, ok := Patchers[config.Id]
	if !ok {
		p.patcherContainer.Hide()
		return
	}

	patcher = patcher.Default()
	if config.Environment != nil {
		patcher = config.Environment
	}

	var items []*widget.FormItem
	items, p.patcherFunc = patcher.ProfileForm()
	p.patcherForm.Items = append(p.patcherForm.Items[:1], items...)

	p.patcherForm.Refresh()
	p.patcherContainer.Show()
}

func NewProfileForm(profile *Profile, window fyne.Window) *ProfileForm {
	form := &ProfileForm{
		name:       widget.NewEntry(),
		serviceUrl: widget.NewEntry(),
		bootForm:   NewBootForm(),
	}

	form.name.PlaceHolder = "My Server"
	form.serviceUrl.PlaceHolder = "https://example.com/MasterIndex"

	patcherHeader := canvas.NewText("Patcher", theme.Color(theme.ColorNameForeground))
	patcherHeader.TextSize = 14

	form.patcherForm = widget.NewForm(
		widget.NewFormItem("Service URL", form.serviceUrl),
	)

	form.patcherContainer = container.NewVBox(
		patcherHeader,
		form.patcherForm,
	)

	form.patcherType = widget.NewSelect(PatcherOptions(), func(s string) {
		if s == "(None)" {
			form.SetPatcherForm(nil)
			return
		}

		var config *PatcherConfig
		if profile != nil && profile.Server.Patcher != nil {
			config = profile.Server.Patcher
		} else {
			config = &PatcherConfig{Id: s}
		}

		form.SetPatcherForm(config)
	})
	form.patcherType.SetSelectedIndex(0)

	xmlOpen := widget.NewButtonWithIcon("", theme.DocumentIcon(), func() {
		dialog := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil {
				dialog.ShowError(fmt.Errorf("failed to open server file: %v", err), window)
				return
			}

			if reader == nil || reader.URI() == nil {
				return
			}

			data, err := os.ReadFile(reader.URI().Path())
			if err != nil {
				dialog.ShowError(err, window)
				return
			}

			serverXml := ServerXml{}
			if err := xml.Unmarshal(data, &serverXml); err != nil {
				dialog.ShowError(fmt.Errorf("malformed server file: %v", err), window)
				return
			}

			if err := form.SetXml(serverXml); err != nil {
				dialog.ShowError(err, window)
			}
		}, window)

		dialog.SetFilter(storage.NewExtensionFileFilter([]string{".xml"}))
		dialog.Show()
	})

	heading := "New Server"
	bootConfig := DefaultBootConfig()

	if profile != nil {
		heading = "Edit Server - " + profile.Name
		bootConfig = profile.Server.BootConfig()
		form.Set(*profile)
	}

	form.bootForm.Set(bootConfig)

	header := canvas.NewText(heading, theme.Color(theme.ColorNameForeground))
	header.TextSize = 16

	bootHeader := canvas.NewText("Boot Config", theme.Color(theme.ColorNameForeground))
	bootHeader.TextSize = 14

	form.Container = container.NewVBox(
		header,
		widget.NewForm(
			widget.NewFormItem("Server XML", container.NewBorder(nil, nil, xmlOpen, nil)),
			widget.NewFormItem("Name", form.name),
			widget.NewFormItem("Patcher", form.patcherType),
		),
		form.patcherContainer,
		widget.NewSeparator(),
		bootHeader,
		form.bootForm.Container,
	)

	return form
}
