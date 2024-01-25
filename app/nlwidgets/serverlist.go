package nlwidgets

import (
	"fmt"

	"fyne.io/fyne/v2/widget"
	"github.com/I-Am-Dench/lu-launcher/app/nlwidgets/muxset"
	"github.com/I-Am-Dench/lu-launcher/resource"
	"github.com/I-Am-Dench/lu-launcher/resource/server"
)

type ServerList struct {
	widget.Select

	servers resource.ServerList

	isDisabled        bool
	currentlyUpdating *muxset.MuxSet[string]
}

func NewServerList(serverList resource.ServerList, changed func(*server.Server)) *ServerList {
	list := new(ServerList)
	list.ExtendBaseWidget(list)

	list.Select.PlaceHolder = "(Select server)"

	list.Select.OnChanged = func(_ string) {
		// If isDisabled and Select.Disabled() are different, then OnChanged is being called through either Disable() or Enable().
		// We only want changed() to be called when an OPTION has been selected or updated.
		if list.isDisabled == list.Select.Disabled() {
			changed(list.SelectedServer())
		}

		list.isDisabled = list.Select.Disabled()
	}

	list.servers = serverList
	list.currentlyUpdating = muxset.New[string]()
	list.isDisabled = false

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

func (list *ServerList) Get(id string) *server.Server {
	return list.servers.Get(id)
}

func (list *ServerList) GetIndex(index int) *server.Server {
	return list.servers.GetIndex(index)
}

func (list *ServerList) Add(server *server.Server) error {
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

func (list *ServerList) Remove(server *server.Server) error {
	if server == nil {
		return fmt.Errorf("fatal remove server error: server is nil")
	}

	if list.currentlyUpdating.Has(server.ID) {
		return fmt.Errorf("cannot remove server: server is currently updating")
	}

	serverIndex := list.servers.Find(server.ID)

	err := list.servers.Remove(server.ID)
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

func (list *ServerList) SelectedServer() *server.Server {
	return list.servers.GetIndex(list.SelectedIndex())
}

func (list *ServerList) MarkAsUpdating(server *server.Server) {
	if server == nil {
		return
	}

	list.currentlyUpdating.Add(server.ID)
}

func (list *ServerList) RemoveAsUpdating(server *server.Server) {
	if server == nil {
		return
	}

	list.currentlyUpdating.Delete(server.ID)
}

func (list *ServerList) Save() error {
	return list.servers.SaveInfos()
}
