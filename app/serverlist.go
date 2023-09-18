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

func NewServerList(serverList resource.ServerList) *ServerList {
	list := new(ServerList)
	list.ExtendBaseWidget(list)

	list.Select.PlaceHolder = "(Select server)"

	list.servers = serverList

	return list
}

func (list *ServerList) refreshInfo() {
	list.SetOptions(list.servers.Names())
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

	list.refreshInfo()
	return nil
}

func (list *ServerList) Remove(server *resource.Server) error {
	if server == nil {
		return fmt.Errorf("fatal remove server error: server is nil")
	}

	err := list.servers.Remove(server.Id)
	if err != nil {
		return err
	}

	list.refreshInfo()
	return nil
}

func (list *ServerList) SetSelectedIndex(index int) {
	list.Select.SetSelectedIndex(index)
	list.refreshInfo()
}

func (list *ServerList) SetSelectedServer(id string) {
	list.SetSelectedIndex(list.servers.Find(id))
	list.refreshInfo()
}

func (list *ServerList) SelectedServer() *resource.Server {
	return list.servers.GetIndex(list.SelectedIndex())
}
