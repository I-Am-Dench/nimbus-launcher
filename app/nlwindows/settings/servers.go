package settings

import (
	"encoding/xml"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/I-Am-Dench/nimbus-launcher/app/forms"
	"github.com/I-Am-Dench/nimbus-launcher/app/nlwidgets"
	"github.com/I-Am-Dench/nimbus-launcher/resource/server"
)

const (
	ExportHeader = "<?xml version=\"1.0\" encoding=\"utf-8\"?>\n\n"
)

type Pager struct {
	containers map[int]*fyne.Container

	Current *fyne.Container
}

func (pager *Pager) Flip(id int) {
	container, ok := pager.containers[id]
	if !ok {
		return
	}

	if container != pager.Current {
		if pager.Current != nil {
			pager.Current.Hide()
		}

		container.Show()
		pager.Current = container
	}
}

func (pager *Pager) Add(container *fyne.Container, id int) {
	container.Hide()
	pager.containers[id] = container
}

func (pager *Pager) BackButton(id int) *widget.Button {
	button := widget.NewButtonWithIcon("Back", theme.NavigateBackIcon(), func() {
		pager.Flip(id)
	})
	button.Importance = widget.LowImportance
	return button
}

func (pager *Pager) Container() *fyne.Container {
	containers := []fyne.CanvasObject{}
	for _, c := range pager.containers {
		containers = append(containers, c)
	}
	return container.NewStack(containers...)
}

const (
	ListPage = iota
	AddPage
	EditPage
)

type ServerSelector struct {
	List     *nlwidgets.ServerList
	Selector *widget.Select
}

func (selector *ServerSelector) Get() *server.Server {
	return selector.List.GetIndex(selector.Selector.SelectedIndex())
}

func (selector *ServerSelector) Add(server *server.Server) error {
	if err := selector.List.Add(server); err != nil {
		return err
	}

	selector.Selector.SetOptions(selector.List.Options)
	selector.Selector.SetSelectedIndex(selector.List.SelectedIndex())
	return nil
}

func (selector *ServerSelector) Remove(server *server.Server) error {
	if err := selector.List.Remove(server); err != nil {
		return err
	}

	selector.Selector.SetOptions(selector.List.Options)
	selector.Selector.SetSelectedIndex(0)
	return nil
}

func (selector *ServerSelector) Update() {
	selector.Selector.SetOptions(selector.List.Options)
	selector.Selector.SetSelectedIndex(selector.List.SelectedIndex())
	selector.List.Refresh()
}

func serverListPage(window fyne.Window, pager *Pager, editServerForm *forms.ServerForm, selector ServerSelector) *fyne.Container {
	heading := canvas.NewText("Servers", theme.ForegroundColor())
	heading.TextSize = 16

	addServerTab := widget.NewButtonWithIcon("Add Server", theme.ContentAddIcon(), func() {
		pager.Flip(AddPage)
	})
	addServerTab.Importance = widget.LowImportance

	menuEdit := fyne.NewMenuItem("Edit", func() {
		server := selector.Get()
		editServerForm.UpdateWith(server)
		pager.Flip(EditPage)
	})
	menuEdit.Icon = theme.DocumentCreateIcon()

	menuRemove := fyne.NewMenuItem("Remove", func() {
		server := selector.Get()
		if server == nil {
			dialog.ShowError(fmt.Errorf("fatal remove error: server is nil"), window)
			return
		}

		dialog.NewCustomConfirm(
			"Remove Server", "Remove", "Cancel", widget.NewLabel(fmt.Sprintf("Remove server configuration '%s'?", server.Name)),
			func(ok bool) {
				if !ok {
					return
				}

				if err := selector.Remove(server); err != nil {
					dialog.ShowError(err, window)
					return
				}

				dialog.ShowInformation("Removed Server", fmt.Sprintf("Server '%s' removed successfully!", server.Name), window)
			},
			window,
		).Show()
	})
	menuRemove.Icon = theme.ContentRemoveIcon()

	menu := widget.NewPopUpMenu(
		fyne.NewMenu("", menuEdit, menuRemove),
		window.Canvas(),
	)

	menuButton := widget.NewButtonWithIcon("", theme.MoreHorizontalIcon(), func() {})
	listContainer := container.NewBorder(nil, nil, nil, menuButton, selector.Selector)

	menuButton.OnTapped = func() {
		menuSize := menu.Size()
		buttonSize := menuButton.Size()

		position := fyne.NewPos(-(menuSize.Width - buttonSize.Width), buttonSize.Height+5)
		menu.ShowAtRelativePosition(position, menuButton)
	}

	return container.NewVBox(
		heading,
		container.NewPadded(
			container.NewVBox(
				container.NewBorder(nil, nil, addServerTab, nil),
				listContainer,
			),
		),
	)
}

func addServerPage(window fyne.Window, pager *Pager, selector ServerSelector) *fyne.Container {
	form := forms.NewServerForm(window, "Add Server")

	addButton := widget.NewButton("Add Server", func() {
		server, err := form.CreateServer()
		if err != nil {
			dialog.ShowError(err, window)
			return
		}

		if err := selector.Add(server); err != nil {
			dialog.ShowError(err, window)
			return
		}

		dialog.ShowInformation("Server Added", fmt.Sprintf("Added '%s' to server list!", server.Name), window)
	})
	addButton.Importance = widget.HighImportance

	return container.NewBorder(
		nil,
		container.NewBorder(nil, nil, pager.BackButton(ListPage), addButton),
		nil, nil,
		container.NewVScroll(
			form.Container(),
		),
	)
}

func PromptExportServer(window fyne.Window, selector ServerSelector) func() {
	return func() {
		dialog := dialog.NewFileSave(func(uc fyne.URIWriteCloser, err error) {
			if uc == nil {
				return
			}

			if err != nil {
				dialog.ShowError(fmt.Errorf("choose file: %v", err), window)
				return
			}

			server := selector.Get()
			if server == nil {
				dialog.ShowError(fmt.Errorf("export server: server is nil"), window)
				return
			}

			serverXML := server.ToXML()
			data, err := xml.MarshalIndent(serverXML, "", "    ")
			if err != nil {
				dialog.ShowError(fmt.Errorf("export server: marshal xml: %v", err), window)
				return
			}

			if _, err := uc.Write(append([]byte(ExportHeader), data...)); err != nil {
				dialog.ShowError(fmt.Errorf("export server: write xml: %v", err), window)
				return
			}

			dialog.ShowInformation("Export Complete", "Exported server configuration successfully!", window)
		}, window)

		dialog.SetFileName("server.xml")
		dialog.SetFilter(storage.NewExtensionFileFilter([]string{".xml"}))
		dialog.Show()
	}
}

func editServerPage(window fyne.Window, pager *Pager, form *forms.ServerForm, selector ServerSelector) *fyne.Container {
	saveButton := widget.NewButton(
		"Save", func() {
			server := selector.Get()
			if server == nil {
				dialog.ShowError(fmt.Errorf("fatal save error: server is nil"), window)
				return
			}

			if err := form.Validate(); err != nil {
				dialog.ShowError(err, window)
				return
			}

			id := server.ID
			version := server.CurrentPatch
			*server = *form.Get()
			server.ID = id
			server.CurrentPatch = version

			if err := server.SaveConfig(); err != nil {
				dialog.ShowError(err, window)
				return
			}

			if err := selector.List.Save(); err != nil {
				dialog.ShowError(err, window)
				return
			}

			selector.Update()

			dialog.ShowInformation("Servers Saved", fmt.Sprintf("Server '%s' saved successfully!", server.Name), window)
		},
	)
	saveButton.Importance = widget.HighImportance

	exportButton := widget.NewButtonWithIcon("Export", theme.DocumentIcon(), PromptExportServer(window, selector))

	return container.NewBorder(
		nil,
		container.NewBorder(nil, nil, pager.BackButton(ListPage), container.NewHBox(exportButton, saveButton)),
		nil, nil,
		container.NewVScroll(
			form.Container(),
		),
	)
}

func NewServersTab(window fyne.Window, list *nlwidgets.ServerList) *container.TabItem {
	pager := &Pager{
		containers: make(map[int]*fyne.Container),
	}

	serverSelector := widget.NewSelect(
		list.Options, func(s string) {},
	)
	serverSelector.SetSelectedIndex(0)

	selector := ServerSelector{
		List:     list,
		Selector: serverSelector,
	}

	editServerForm := forms.NewServerForm(window, "Edit Server")

	pager.Add(serverListPage(window, pager, editServerForm, selector), ListPage)
	pager.Add(addServerPage(window, pager, selector), AddPage)
	pager.Add(editServerPage(window, pager, editServerForm, selector), EditPage)
	pager.Flip(ListPage)

	return container.NewTabItem(
		"Servers", container.NewPadded(pager.Container()),
	)
}
