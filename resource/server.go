package resource

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/I-Am-Dench/lu-launcher/luconfig"
)

type Server struct {
	Name string `json:"name"`
	Boot string `json:"boot"`

	Config *luconfig.LUConfig `json:"-"`
}

func NewServer(name string, config *luconfig.LUConfig) (*Server, error) {
	server := new(Server)
	server.Name = name
	server.Config = config

	err := server.SaveConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create new server: %v", err)
	}

	return server, nil
}

func (server *Server) DeleteConfig() error {
	return os.Remove(Of(settingsDir, serversDir, server.Boot))
}

func (server *Server) SaveConfig() error {
	bootTempName := server.Boot
	if len(server.Boot) == 0 {
		bootTempName = fmt.Sprintf("boot_%d.cfg", time.Now().Unix())
		server.Boot = bootTempName
	}

	data, err := luconfig.Marshal(server.Config)
	if err != nil {
		return fmt.Errorf("cannot marshal boot.cfg: %v", err)
	}

	err = os.WriteFile(Of(settingsDir, serversDir, bootTempName), data, 0755)
	if err != nil {
		return fmt.Errorf("cannot save boot.cfg: %v", err)
	}

	return nil
}

func (server *Server) LoadConfig() error {
	server.Config = luconfig.New()

	if len(server.Boot) == 0 {
		return nil
	}

	data, err := os.ReadFile(Of(settingsDir, serversDir, server.Boot))
	if err != nil {
		return fmt.Errorf("cannot load boot.cfg: %v", err)
	}

	err = luconfig.Unmarshal(data, server.Config)
	if err != nil {
		return fmt.Errorf("cannot unmarshal boot.cfg: %v", err)
	}

	return nil
}

type ServerList struct {
	servers []*Server
}

func (servers *ServerList) List() []*Server {
	return servers.servers
}

func (servers *ServerList) Size() int {
	return len(servers.servers)
}

func (servers *ServerList) Names() []string {
	names := []string{}
	for _, server := range servers.servers {
		names = append(names, server.Name)
	}
	return names
}

func (servers *ServerList) Get(index int) *Server {
	if 0 <= index && index < servers.Size() {
		return servers.servers[index]
	}
	return nil
}

func (servers *ServerList) Add(server *Server) error {
	servers.servers = append(servers.servers, server)
	return servers.SaveInfo()
}

func (servers *ServerList) Remove(index int) error {
	servers.servers = append(servers.servers[:index], servers.servers[index+1:]...)
	return servers.SaveInfo()
}

func (servers *ServerList) SaveInfo() error {
	data, err := json.MarshalIndent(servers.servers, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to save servers: %v", err)
	}

	err = os.WriteFile(Of(settingsDir, "servers.json"), data, 0755)
	if err != nil {
		return fmt.Errorf("failed to save servers: %v", err)
	}

	return nil
}

func (servers *ServerList) Load() error {
	data, err := os.ReadFile(Of(settingsDir, "servers.json"))
	if err != nil {
		return fmt.Errorf("cannot read servers.json: %v", err)
	}

	err = json.Unmarshal(data, &servers.servers)
	if err != nil {
		return fmt.Errorf("cannot unmarshal servers.json: %v", err)
	}

	for _, server := range servers.servers {
		err := server.LoadConfig()
		if err != nil {
			log.Printf("load servers error: %v\n", err)
		}
	}

	return nil
}
