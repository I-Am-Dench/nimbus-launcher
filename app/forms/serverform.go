package forms

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/I-Am-Dench/nimbus-launcher/ldf"
	"github.com/I-Am-Dench/nimbus-launcher/resource"
	"github.com/I-Am-Dench/nimbus-launcher/resource/server"
)

type ServerForm struct {
	container *fyne.Container

	serverXMLFile *widget.Label

	title         *widget.Entry
	patchToken    *widget.Entry
	patchProtocol *widget.Select

	bootForm *BootForm
}

func NewServerForm(window fyne.Window, heading string) *ServerForm {
	form := new(ServerForm)

	infoHeading := canvas.NewText(heading, theme.ForegroundColor())
	infoHeading.TextSize = 16

	bootHeading := canvas.NewText("boot.cfg", theme.ForegroundColor())
	bootHeading.TextSize = 16

	form.serverXMLFile = widget.NewLabel("")
	form.serverXMLFile.Truncation = fyne.TextTruncateEllipsis

	form.title = widget.NewEntry()
	form.title.PlaceHolder = "My Server"

	form.patchToken = widget.NewPasswordEntry()

	form.patchProtocol = widget.NewSelect(
		[]string{"(None)", "http", "https"}, func(s string) {
			if s == "(None)" {
				form.patchProtocol.ClearSelected()
			}
		},
	)
	form.patchProtocol.PlaceHolder = "(None)"

	form.bootForm = NewBootForm(window)

	serverXMLOpen := widget.NewButtonWithIcon("", theme.FileIcon(), form.PromptServerXMLFile(window))

	form.container = container.NewVBox(
		infoHeading,
		widget.NewForm(
			widget.NewFormItem("Server XML", container.NewBorder(nil, nil, serverXMLOpen, nil, form.serverXMLFile)),
			widget.NewFormItem("Name", form.title),
			widget.NewFormItem("Patch Token", form.patchToken),
			widget.NewFormItem("Patch Protocol", form.patchProtocol),
		),
		widget.NewSeparator(),
		bootHeading,
		form.bootForm.Container(),
	)

	return form
}

func (form *ServerForm) PromptServerXMLFile(window fyne.Window) func() {
	return func() {
		dialog := dialog.NewFileOpen(func(uc fyne.URIReadCloser, err error) {
			if err != nil {
				dialog.ShowError(fmt.Errorf("error when opening server.xml file: %v", err), window)
				return
			}

			if uc == nil || uc.URI() == nil {
				return
			}
			form.serverXMLFile.SetText(uc.URI().Path())

			server, err := server.LoadXML(uc.URI().Path())
			if err != nil {
				dialog.ShowError(err, window)
				return
			}

			form.title.SetText(server.Name)
			form.patchToken.SetText(server.Patch.Token)
			form.patchProtocol.SetSelected(server.Patch.Protocol)

			bootConfig := ldf.BootConfig{}
			err = ldf.Unmarshal([]byte(server.Boot.Text), &bootConfig)
			if err != nil {
				dialog.ShowError(err, window)
				return
			}

			form.bootForm.UpdateWith(&bootConfig)
		}, window)

		dialog.SetFilter(storage.NewExtensionFileFilter([]string{".xml"}))
		dialog.Show()
	}
}

func (form *ServerForm) CreateServer() (*server.Server, error) {
	err := form.Validate()
	if err != nil {
		return nil, err
	}

	return resource.CreateServer(server.Config{
		Name:          form.title.Text,
		PatchToken:    form.patchToken.Text,
		PatchProtocol: form.patchProtocol.Selected,
		Config:        form.bootForm.GetConfig(),
	})
}

func (form *ServerForm) UpdateWith(server *server.Server) {
	if server == nil {
		return
	}

	form.title.SetText(server.Name)
	form.patchToken.SetText(server.PatchToken)
	form.patchProtocol.SetSelected(server.PatchProtocol)

	form.bootForm.UpdateWith(server.Config)
}

func (form *ServerForm) Get() *server.Server {
	return resource.NewServer(server.Config{
		Name:          form.title.Text,
		PatchToken:    form.patchToken.Text,
		PatchProtocol: form.patchProtocol.Selected,
		Config:        form.bootForm.GetConfig(),
	})
}

func (form *ServerForm) Validate() error {
	if len(form.title.Text) <= 0 {
		return fmt.Errorf("name cannot be empty")
	}

	return nil
}

func (form *ServerForm) Container() *fyne.Container {
	return form.container
}
