package luwidgets

import (
	"fmt"

	"fyne.io/fyne/v2/widget"
	"github.com/I-Am-Dench/lu-launcher/app/muxset"
	"github.com/I-Am-Dench/lu-launcher/resource"
)

type ServerList struct {
	widget.Select

	servers resource.ServerList

	currentlyUpdating *muxset.MuxSet[string]
}

func NewServerList(serverList resource.ServerList, changed func(*resource.Server)) *ServerList {
	list := new(ServerList)
	list.ExtendBaseWidget(list)

	list.Select.PlaceHolder = "(Select server)"

	list.Select.OnChanged = func(_ string) {
		if list.Select.Disabled() {
			return
		}

		changed(list.SelectedServer())
	}

	list.servers = serverList
	list.currentlyUpdating = muxset.New[string]()

	list.Refresh()

	return list
}

func (list *ServerList) Refresh() {
	// SetOptions resets the index to -1, so we get the current index in order to refresh the current selection afterwards
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

	if list.currentlyUpdating.Has(server.Id) {
		return fmt.Errorf("cannot remove server: server is currently updating")
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

func (list *ServerList) MarkAsUpdating(server *resource.Server) {
	if server == nil {
		return
	}

	list.currentlyUpdating.Add(server.Id)
}

func (list *ServerList) RemoveAsUpdating(server *resource.Server) {
	if server == nil {
		return
	}

	list.currentlyUpdating.Delete(server.Id)
}

func (list *ServerList) Save() error {
	return list.servers.SaveInfos()
}
