package app

import (
	"encoding/xml"
	"fmt"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/I-Am-Dench/lu-launcher/app/forms"
	"github.com/I-Am-Dench/lu-launcher/luwidgets"
)

const (
	EXPORT_HEADER = "<?xml version=\"1.0\" encoding=\"utf-8\"?>\n\n"
)

type ServersPage struct {
	container *fyne.Container

	serverList *widget.Select

	buttons *fyne.Container

	addServers  *fyne.Container
	editServers *fyne.Container
}

func NewServersPage(window fyne.Window, list *luwidgets.ServerList) *ServersPage {
	page := new(ServersPage)

	page.serverList = widget.NewSelect(
		list.Options, func(s string) {},
	)
	page.serverList.SetSelectedIndex(0)

	addServerForm := forms.NewServerForm(window)
	page.addServers = page.addServerPage(addServerForm, window, list)
	page.addServers.Hide()

	editServerForm := forms.NewServerForm(window)
	page.editServers = page.editServerPage(editServerForm, window, list)
	page.editServers.Hide()

	heading := canvas.NewText("Servers", color.White)
	heading.TextSize = 16

	addServerTab := widget.NewButtonWithIcon("Add Server", theme.ContentAddIcon(), func() {
		page.buttons.Hide()
		page.addServers.Show()
	})
	addServerTab.Importance = widget.LowImportance

	menuEdit := fyne.NewMenuItem("Edit", func() {
		server := list.GetIndex(page.serverList.SelectedIndex())
		editServerForm.UpdateWith(server)

		page.buttons.Hide()
		page.editServers.Show()
	})
	menuEdit.Icon = theme.DocumentCreateIcon()

	menuRemove := fyne.NewMenuItem("Remove", func() {
		server := list.GetIndex(page.serverList.SelectedIndex())
		if server == nil {
			dialog.ShowError(fmt.Errorf("fatal remove error: server is nil"), window)
			return
		}

		confirm := dialog.NewCustomConfirm(
			"Remove Server", "Remove", "Cancel", widget.NewLabel(fmt.Sprintf("Remove server configuration '%s'?", server.Name)),
			func(ok bool) {
				if !ok {
					return
				}

				err := list.Remove(server)
				if err != nil {
					dialog.ShowError(err, window)
					return
				}

				page.serverList.SetOptions(list.Options)
				page.serverList.SetSelectedIndex(0)
				dialog.ShowInformation("Remove Server", fmt.Sprintf("Server '%s' removed successfully!", server.Name), window)
			},
			window,
		)

		confirm.Show()
	})
	menuRemove.Icon = theme.ContentRemoveIcon()

	menu := widget.NewPopUpMenu(
		fyne.NewMenu("", menuEdit, menuRemove),
		window.Canvas(),
	)

	menuButton := widget.NewButtonWithIcon("", theme.MoreHorizontalIcon(), func() {})
	listContainer := container.NewBorder(nil, nil, nil, menuButton, page.serverList)

	menuButton.OnTapped = func() {
		menuSize := menu.Size()
		buttonSize := menuButton.Size()

		position := fyne.NewPos(-(menuSize.Width - buttonSize.Width), buttonSize.Height+5)
		menu.ShowAtRelativePosition(position, menuButton)
	}

	page.buttons = container.NewVBox(
		heading,
		container.NewPadded(
			container.NewVBox(
				container.NewBorder(nil, nil, addServerTab, nil),
				listContainer,
			),
		),
	)

	page.container = container.NewStack(
		page.buttons,
		page.addServers,
		page.editServers,
	)

	return page
}

func (page *ServersPage) addServerPage(form *forms.ServerForm, window fyne.Window, list *luwidgets.ServerList) *fyne.Container {
	addButton := widget.NewButton("Add Server", func() {
		server, err := form.CreateServer()
		if err != nil {
			dialog.ShowError(err, window)
			return
		}

		err = list.Add(server)
		if err != nil {
			dialog.ShowError(err, window)
			return
		}

		page.serverList.SetOptions(list.Options)
		page.serverList.SetSelectedIndex(page.serverList.SelectedIndex())
		dialog.ShowInformation("Server Added", fmt.Sprintf("Added '%s' to server list!", server.Name), window)
	})
	addButton.Importance = widget.HighImportance

	return container.NewBorder(
		nil,
		container.NewBorder(nil, nil, BackButton(func() {
			page.addServers.Hide()
			page.buttons.Show()
		}), addButton),
		nil, nil,
		container.NewVScroll(
			form.Container(),
		),
	)
}

func (page *ServersPage) editServerPage(form *forms.ServerForm, window fyne.Window, list *luwidgets.ServerList) *fyne.Container {
	saveButton := widget.NewButton(
		"Save", func() {
			server := list.GetIndex(page.serverList.SelectedIndex())
			if server == nil {
				dialog.ShowError(fmt.Errorf("fatal save error: server is nil"), window)
				return
			}

			if err := form.Validate(); err != nil {
				dialog.ShowError(err, window)
				return
			}

			id := server.Id
			version := server.CurrentPatch
			*server = *form.Get()
			server.Id = id
			server.CurrentPatch = version

			err := server.SaveConfig()
			if err != nil {
				dialog.ShowError(err, window)
				return
			}

			err = list.Save()
			if err != nil {
				dialog.ShowError(err, window)
				return
			}

			page.serverList.SetOptions(list.Options)
			page.serverList.SetSelectedIndex(page.serverList.SelectedIndex())

			list.Refresh()

			dialog.ShowInformation("Servers Saved", fmt.Sprintf("Server '%s' saved successfully!", server.Name), window)
		},
	)
	saveButton.Importance = widget.HighImportance

	exportButton := widget.NewButtonWithIcon(
		"Export", theme.DocumentIcon(),
		func() {
			dialog := dialog.NewFileSave(func(uc fyne.URIWriteCloser, err error) {
				if uc == nil {
					return
				}

				if err != nil {
					dialog.ShowError(fmt.Errorf("error when choosing file: %v", err), window)
					return
				}

				server := list.GetIndex(page.serverList.SelectedIndex())
				if server == nil {
					dialog.ShowError(fmt.Errorf("fatal export server error: server is nil"), window)
					return
				}

				serverXML := server.ToXML()
				data, err := xml.MarshalIndent(serverXML, "", "    ")
				if err != nil {
					dialog.ShowError(fmt.Errorf("cannot marshal server.xml: %v", err), window)
					return
				}

				_, err = uc.Write(append([]byte(EXPORT_HEADER), data...))
				if err != nil {
					dialog.ShowError(fmt.Errorf("error when writing to server.xml: %v", err), window)
					return
				}

				dialog.ShowInformation("Export Complete", "Exported server configuration successfully!", window)
			}, window)
			dialog.SetFileName("server.xml")
			dialog.SetFilter(storage.NewExtensionFileFilter([]string{".xml"}))
			dialog.Show()
		},
	)

	return container.NewBorder(
		nil,
		container.NewBorder(nil, nil, BackButton(func() {
			page.editServers.Hide()
			page.buttons.Show()
		}), container.NewHBox(exportButton, saveButton)),
		nil, nil,
		container.NewVScroll(
			form.Container(),
		),
	)
}

func (page *ServersPage) Container() *fyne.Container {
	return page.container
}
