package resource

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

type ServerListContainer interface {
	ServerNames() []string
	GetServer(int) *Server

	AddServer(*Server) error
	RemoveServer(*Server) error

	SaveServers() error
}

type ServerList struct {
	list []*Server
}

func (servers *ServerList) SaveInfos() error {
	data, err := json.MarshalIndent(servers.list, "", "    ")
	if err != nil {
		return fmt.Errorf("cannot save server infos: %v", err)
	}

	err = os.WriteFile(Of(settingsDir, "servers.json"), data, 0755)
	if err != nil {
		return fmt.Errorf("cannot save server infos: %v", err)
	}

	return nil
}

func (servers *ServerList) Load() error {
	data, err := os.ReadFile(Of(settingsDir, "servers.json"))
	if err != nil {
		return fmt.Errorf("cannot read servers.json: %v", err)
	}

	err = json.Unmarshal(data, &servers.list)
	if err != nil {
		return fmt.Errorf("cannot unmarshal servers.json: %v", err)
	}

	for _, server := range servers.list {
		err := server.LoadConfig()
		if err != nil {
			log.Printf("load servers error: %v\n", err)
		}
	}

	return nil
}

func (servers *ServerList) Size() int {
	return len(servers.list)
}

func (servers *ServerList) List() []*Server {
	return servers.list
}

func (servers *ServerList) Names() []string {
	names := []string{}
	for _, server := range servers.list {
		names = append(names, server.Name)
	}
	return names
}

func (servers *ServerList) Get(id string) *Server {
	for _, server := range servers.list {
		if server.Id == id {
			return server
		}
	}
	return nil
}

func (servers *ServerList) GetIndex(index int) *Server {
	if 0 <= index && index < servers.Size() {
		return servers.list[index]
	}
	return nil
}

func (servers *ServerList) Find(id string) int {
	for i, server := range servers.list {
		if server.Id == id {
			return i
		}
	}
	return -1
}

func (servers *ServerList) Add(server *Server) error {
	servers.list = append(servers.list, server)
	return servers.SaveInfos()
}

func (servers *ServerList) Remove(id string) error {
	index := servers.Find(id)
	if index < 0 {
		return nil
	}

	server := servers.GetIndex(index)
	err := server.DeleteConfig()
	if err != nil {
		log.Printf("Unable to remove boot.cfg for '%s': %v\n", server.Name, err)
	}

	servers.list = append(servers.list[:index], servers.list[index+1:]...)
	return servers.SaveInfos()
}
