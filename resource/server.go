package resource

import (
	"fmt"
	"os"
	"time"

	"github.com/I-Am-Dench/lu-launcher/luconfig"
)

type Server struct {
	Id           string `json:"id"`
	Name         string `json:"name"`
	Boot         string `json:"boot"`
	PatchServer  string `json:"patchServer"`
	CurrentPatch string `json:"currentPatch"`

	Config *luconfig.LUConfig `json:"-"`
}

func NewServer(name, patchServer string, config *luconfig.LUConfig) *Server {
	server := new(Server)
	server.Id = fmt.Sprint(time.Now().Unix())
	server.Name = name
	server.Config = config
	return server
}

func CreateServer(name, patchServer string, config *luconfig.LUConfig) (*Server, error) {
	server := NewServer(name, patchServer, config)
	return server, server.SaveConfig()
}

func (server *Server) SaveConfig() error {
	bootName := server.Boot
	if len(server.Boot) == 0 {
		bootName = fmt.Sprintf("boot_%s.cfg", server.Id)
		server.Boot = bootName
	}

	data, err := luconfig.Marshal(server.Config)
	if err != nil {
		return fmt.Errorf("cannot marshal boot.cfg: %v", err)
	}

	err = os.WriteFile(Of(settingsDir, serversDir, bootName), data, 0755)
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

func (server *Server) DeleteConfig() error {
	return os.Remove(server.BootPath())
}

func (server *Server) BootPath() string {
	return Of(settingsDir, serversDir, server.Boot)
}
