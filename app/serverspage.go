package app

import (
	"fmt"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/I-Am-Dench/lu-launcher/app/forms"
	"github.com/I-Am-Dench/lu-launcher/resource"
)

// type serverList interface {
// 	ServerNames() []string
// 	GetServer(int) *resource.Server
// 	SetServer(int, *resource.Server)
// 	SaveServers() error
// 	AddServer(*resource.Server) error
// }

type ServersPage struct {
	container *fyne.Container

	buttons *fyne.Container

	addServers  *fyne.Container
	editServers *fyne.Container
}

func NewServersPage(window fyne.Window, list resource.ServerListContainer) *ServersPage {
	page := new(ServersPage)

	serverList := widget.NewSelect(
		list.ServerNames(), func(s string) {},
	)
	serverList.SetSelectedIndex(0)

	addServerForm := forms.NewServerForm(window)
	addServerButton := widget.NewButton("Add Server", func() {
		server, err := addServerForm.CreateServer()
		if err != nil {
			dialog.ShowError(err, window)
			return
		}

		err = list.AddServer(server)
		if err != nil {
			dialog.ShowError(err, window)
			return
		}

		serverList.SetOptions(list.ServerNames())
		serverList.SetSelectedIndex(serverList.SelectedIndex())
		dialog.ShowInformation("Server Added", fmt.Sprintf("Added '%s' to server list!", server.Name), window)
	})
	addServerButton.Importance = widget.HighImportance

	page.addServers = container.NewBorder(
		nil,
		container.NewBorder(nil, nil, BackButton(func() {
			page.addServers.Hide()
			page.buttons.Show()
		}), addServerButton),
		nil, nil,
		container.NewVScroll(
			addServerForm.Container(),
		),
	)
	page.addServers.Hide()

	editServerForm := forms.NewServerForm(window)
	editServerButton := widget.NewButton(
		"Save", func() {
			server := list.GetServer(serverList.SelectedIndex())
			if server == nil {
				dialog.ShowError(fmt.Errorf("fatal save error: server is nil"), window)
				return
			}

			id := server.Id
			*server = *editServerForm.Get()
			server.Id = id

			err := server.SaveConfig()
			if err != nil {
				dialog.ShowError(err, window)
				return
			}

			err = list.SaveServers()
			if err != nil {
				dialog.ShowError(err, window)
				return
			}

			serverList.SetOptions(list.ServerNames())
			serverList.SetSelectedIndex(serverList.SelectedIndex())
			dialog.ShowInformation("Servers Saved", fmt.Sprintf("Server '%s' saved successfully!", server.Name), window)
		},
	)
	editServerButton.Importance = widget.HighImportance

	page.editServers = container.NewBorder(
		nil,
		container.NewBorder(nil, nil, BackButton(func() {
			page.editServers.Hide()
			page.buttons.Show()
		}), editServerButton),
		nil, nil,
		container.NewVScroll(
			editServerForm.Container(),
		),
	)
	page.editServers.Hide()

	heading := canvas.NewText("Servers", color.White)
	heading.TextSize = 16

	addServerTab := widget.NewButtonWithIcon("Add Server", theme.ContentAddIcon(), func() {
		page.buttons.Hide()
		page.addServers.Show()
	})
	addServerTab.Importance = widget.LowImportance
	addServerTab.Alignment = widget.ButtonAlignLeading

	editServerTab := widget.NewButtonWithIcon(
		"", theme.DocumentCreateIcon(),
		func() {
			server := list.GetServer(serverList.SelectedIndex())
			editServerForm.UpdateWith(server)

			page.buttons.Hide()
			page.editServers.Show()
		},
	)

	removeServerButton := widget.NewButtonWithIcon(
		"Remove Server", theme.ContentRemoveIcon(),
		func() {
			server := list.GetServer(serverList.SelectedIndex())
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

					err := list.RemoveServer(server)
					if err != nil {
						dialog.ShowError(err, window)
						return
					}

					serverList.SetOptions(list.ServerNames())
					serverList.SetSelectedIndex(0)
					dialog.ShowInformation("Remove Server", fmt.Sprintf("Server '%s' removed successfully!", server.Name), window)
				},
				window,
			)

			confirm.Show()
		},
	)
	removeServerButton.Importance = widget.LowImportance

	page.buttons = container.NewVBox(
		heading,
		container.NewBorder(nil, nil, addServerTab, nil),
		container.NewBorder(
			nil, nil, nil,
			editServerTab,
			serverList,
		),
		container.NewBorder(nil, nil, removeServerButton, nil),
	)

	page.container = container.NewStack(
		page.buttons,
		page.addServers,
		page.editServers,
	)

	return page
}

func (page *ServersPage) Container() *fyne.Container {
	return page.container
}
