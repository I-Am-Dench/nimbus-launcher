package resource

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/I-Am-Dench/lu-launcher/resource/server"
)

type ServerList struct {
	list []*server.Server
}

func (list *ServerList) Path() string {
	return filepath.Join(settingsDir, "servers.json")
}

func (list *ServerList) SaveInfos() error {
	data, err := json.MarshalIndent(list.list, "", "    ")
	if err != nil {
		return fmt.Errorf("cannot save server infos: %v", err)
	}

	err = os.WriteFile(list.Path(), data, 0755)
	if err != nil {
		return fmt.Errorf("cannot save server infos: %v", err)
	}

	return nil
}

func (list *ServerList) Load() error {
	data, err := os.ReadFile(list.Path())
	if err != nil {
		return fmt.Errorf("cannot read servers.json: %v", err)
	}

	err = json.Unmarshal(data, &list.list)
	if err != nil {
		return fmt.Errorf("cannot unmarshal servers.json: %v", err)
	}

	for _, server := range list.list {
		err := server.LoadConfig(settingsDir, "patches")
		if err != nil {
			log.Printf("load servers error: %v", err)
		}
	}

	return nil
}

func (list *ServerList) Size() int {
	return len(list.list)
}

func (list *ServerList) Names() []string {
	names := []string{}
	for _, server := range list.list {
		names = append(names, server.Name)
	}
	return names
}

func (list *ServerList) Get(id string) *server.Server {
	for _, server := range list.list {
		if server.ID == id {
			return server
		}
	}
	return nil
}

func (list *ServerList) GetIndex(index int) *server.Server {
	if 0 <= index && index < list.Size() {
		return list.list[index]
	}
	return nil
}

func (list *ServerList) Find(id string) int {
	for i, server := range list.list {
		for server.ID == id {
			return i
		}
	}
	return -1
}

func (list *ServerList) Add(server *server.Server) error {
	if server == nil {
		panic(fmt.Errorf("server list: add: server cannot be nil"))
	}

	list.list = append(list.list, server)
	return list.SaveInfos()
}

func (list *ServerList) Remove(id string) error {
	index := list.Find(id)
	if index < 0 {
		return nil
	}

	server := list.GetIndex(index)
	err := server.DeleteConfig()
	if err != nil {
		log.Printf("Unable to remove boot.cfg for \"%s\": %v", server.Name, err)
	}

	list.list = append(list.list[:index], list.list[index+1:]...)
	return list.SaveInfos()
}
