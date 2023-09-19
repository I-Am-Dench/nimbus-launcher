package app

import (
	"fmt"

	"fyne.io/fyne/v2/widget"
	"github.com/I-Am-Dench/lu-launcher/resource"
)

type ServerList struct {
	widget.Select

	servers resource.ServerList
}

func NewServerList(serverList resource.ServerList, changed func(*resource.Server)) *ServerList {
	list := new(ServerList)
	list.ExtendBaseWidget(list)

	list.Select.PlaceHolder = "(Select server)"

	list.Select.OnChanged = func(_ string) {
		changed(list.SelectedServer())
	}

	list.servers = serverList
	list.Refresh()

	return list
}

func (list *ServerList) Refresh() {
	// SetOptions resets the index to -1, so we get the current index to refresh the current selection after
	selectedIndex := list.SelectedIndex()
	list.SetOptions(list.servers.Names())
	list.SetSelectedIndex(selectedIndex)

	list.Select.Refresh()
}

func (list *ServerList) Get(id string) *resource.Server {
	return list.servers.Get(id)
}

func (list *ServerList) GetIndex(index int) *resource.Server {
	return list.servers.GetIndex(index)
}

func (list *ServerList) Add(server *resource.Server) error {
	if server == nil {
		return fmt.Errorf("fatal add server error: server is nil")
	}

	err := list.servers.Add(server)
	if err != nil {
		return err
	}

	list.Refresh()
	return nil
}

func (list *ServerList) Remove(server *resource.Server) error {
	if server == nil {
		return fmt.Errorf("fatal remove server error: server is nil")
	}

	serverIndex := list.servers.Find(server.Id)

	err := list.servers.Remove(server.Id)
	if err != nil {
		return err
	}

	if serverIndex == list.SelectedIndex() {
		list.ClearSelected()
	}

	list.Refresh()
	return nil
}

func (list *ServerList) SetSelectedIndex(index int) {
	list.Select.SetSelectedIndex(index)
}

func (list *ServerList) SetSelectedServer(id string) {
	list.SetSelectedIndex(list.servers.Find(id))
}

func (list *ServerList) SelectedServer() *resource.Server {
	return list.servers.GetIndex(list.SelectedIndex())
}

func (list *ServerList) Save() error {
	return list.servers.SaveInfos()
}
