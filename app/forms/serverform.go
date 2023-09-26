package forms

import (
	"fmt"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/I-Am-Dench/lu-launcher/luconfig"
	"github.com/I-Am-Dench/lu-launcher/resource"
)

type ServerForm struct {
	container *fyne.Container

	title         *widget.Entry
	patchToken    *widget.Entry
	patchProtocol *widget.Select

	bootForm *BootForm
}

func NewServerForm(window fyne.Window, heading string) *ServerForm {
	form := new(ServerForm)

	infoHeading := canvas.NewText(heading, color.White)
	infoHeading.TextSize = 16

	bootHeading := canvas.NewText("boot.cfg", color.White)
	bootHeading.TextSize = 16

	serverXML := widget.NewLabel("")
	serverXML.Truncation = fyne.TextTruncateEllipsis

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

	serverXMLOpen := widget.NewButtonWithIcon(
		"", theme.FileIcon(),
		func() {
			fileDialog := dialog.NewFileOpen(func(uc fyne.URIReadCloser, err error) {
				if err != nil {
					dialog.ShowError(fmt.Errorf("error when opening server.xml file: %v", err), window)
					return
				}

				if uc == nil || uc.URI() == nil {
					return
				}
				serverXML.SetText(uc.URI().Path())

				server, err := resource.LoadXML(uc.URI().Path())
				if err != nil {
					dialog.ShowError(err, window)
					return
				}

				form.title.SetText(server.Name)
				form.patchToken.SetText(server.Patch.Token)
				form.patchProtocol.SetSelected(server.Patch.Protocol)

				bootConfig := luconfig.LUConfig{}
				err = luconfig.Unmarshal([]byte(server.Boot.Text), &bootConfig)
				if err != nil {
					dialog.ShowError(err, window)
					return
				}

				form.bootForm.UpdateWith(&bootConfig)
			}, window)
			fileDialog.SetFilter(storage.NewExtensionFileFilter([]string{".xml"}))
			fileDialog.Show()
		},
	)

	form.container = container.NewVBox(
		infoHeading,
		widget.NewForm(
			widget.NewFormItem("Server XML", container.NewBorder(nil, nil, serverXMLOpen, nil, serverXML)),
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

func (form *ServerForm) CreateServer() (*resource.Server, error) {
	err := form.Validate()
	if err != nil {
		return nil, err
	}

	return resource.CreateServer(form.title.Text, form.patchToken.Text, form.patchProtocol.Selected, form.bootForm.GetConfig())
}

func (form *ServerForm) UpdateWith(server *resource.Server) {
	if server == nil {
		return
	}

	form.title.SetText(server.Name)
	form.patchToken.SetText(server.PatchToken)
	form.patchProtocol.SetSelected(server.PatchProtocol)

	form.bootForm.UpdateWith(server.Config)
}

func (form *ServerForm) Get() *resource.Server {
	return resource.NewServer(form.title.Text, form.patchToken.Text, form.patchProtocol.Selected, form.bootForm.GetConfig())
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
