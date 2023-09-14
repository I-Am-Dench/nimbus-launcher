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

	title *widget.Entry
	// patchServer *widget.Entry

	bootForm *BootForm
}

func NewServerForm(window fyne.Window) *ServerForm {
	form := new(ServerForm)

	infoHeading := canvas.NewText("Server Info", color.White)
	infoHeading.TextSize = 16

	bootHeading := canvas.NewText("boot.cfg", color.White)
	bootHeading.TextSize = 16

	serverXML := widget.NewLabel("")

	form.title = widget.NewEntry()
	form.title.PlaceHolder = "My Server"

	// form.patchServer = widget.NewEntry()

	form.bootForm = NewBootForm()

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
				// form.patchServer.SetText(server.PatchServer)

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
			widget.NewFormItem("Server XML", container.NewHBox(serverXMLOpen, serverXML)),
			widget.NewFormItem("Name", form.title),
			// widget.NewFormItem("Patch Server", form.patchServer),
		),
		widget.NewSeparator(),
		bootHeading,
		form.bootForm.Container(),
	)

	return form
}

func (form *ServerForm) CreateServer() (*resource.Server, error) {
	return resource.CreateServer(form.title.Text, form.bootForm.GetConfig())
}

func (form *ServerForm) UpdateWith(server *resource.Server) {
	if server == nil {
		return
	}

	form.title.SetText(server.Name)
	// form.patchServer.SetText(server.PatchServer)

	form.bootForm.UpdateWith(server.Config)
}

func (form *ServerForm) Get() *resource.Server {
	return resource.NewServer(form.title.Text, form.bootForm.GetConfig())
}

func (form *ServerForm) Container() *fyne.Container {
	return form.container
}
