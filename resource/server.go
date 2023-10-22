package resource

import (
	"encoding/xml"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/I-Am-Dench/lu-launcher/luconfig"
)

const (
	HEADER_PATCH_TOKEN = "TPP-Token"
)

type Server struct {
	Id            string `json:"id"`
	Name          string `json:"name"`
	Boot          string `json:"boot"`
	PatchToken    string `json:"patchToken"`
	PatchProtocol string `json:"patchProtocol"`
	CurrentPatch  string `json:"currentPatch"`

	Config *luconfig.LUConfig `json:"-"`

	hasPatchesList bool          `json:"-"`
	patchesList    PatchVersions `json:"-"`

	pendingUpdate bool `json:"-"`
}

func NewServer(name, patchToken, patchProtocol string, config *luconfig.LUConfig) *Server {
	server := new(Server)
	server.Id = fmt.Sprint(time.Now().Unix())
	server.Name = name
	server.PatchToken = patchToken
	server.PatchProtocol = patchProtocol
	server.Config = config
	return server
}

func CreateServer(name, patchToken, patchProtocol string, config *luconfig.LUConfig) (*Server, error) {
	server := NewServer(name, patchToken, patchProtocol, config)
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

	err = os.WriteFile(filepath.Join(settingsDir, serversDir, bootName), data, 0755)
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

	data, err := os.ReadFile(filepath.Join(settingsDir, serversDir, server.Boot))
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
	return filepath.Join(settingsDir, serversDir, server.Boot)
}

func (server *Server) PatchServerHost() string {
	return fmt.Sprint(server.PatchProtocol, "://", server.Config.PatchServerIP, ":", server.Config.PatchServerPort)
}

func (server *Server) PatchServerUrl(elem ...string) (string, error) {
	path := []string{server.Config.PatchServerDir}
	return url.JoinPath(server.PatchServerHost(), append(path, elem...)...)
}

func (server *Server) PatchServerRequest(elem ...string) (*http.Response, error) {
	url, err := server.PatchServerUrl(elem...)
	if err != nil {
		return nil, fmt.Errorf("could not create patch server URL with \"%s\": %v", server.PatchServerHost(), err)
	}

	log.Printf("Patch server request: %s\n", url)
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	if len(server.PatchToken) > 0 {
		request.Header.Set(HEADER_PATCH_TOKEN, server.PatchToken)
	}

	client := http.Client{}
	return client.Do(request)
}

func (server *Server) ToXML() ServerXML {
	data, _ := luconfig.Marshal(server.Config)
	return ServerXML{
		Name: server.Name,
		Boot: struct {
			Text string `xml:",innerxml"`
		}{
			Text: string(data),
		},
		Patch: struct {
			XMLName  xml.Name `xml:"patch"`
			Token    string   `xml:"token"`
			Protocol string   `xml:"protocol"`
		}{
			Token:    server.PatchToken,
			Protocol: server.PatchProtocol,
		},
	}
}

func (server *Server) PatchVersions() (PatchVersions, bool) {
	return server.patchesList, server.hasPatchesList
}

func (server *Server) SetPatchVersions(version PatchVersions) {
	server.patchesList = version
	server.hasPatchesList = true
}

func (server *Server) PendingUpdate() bool {
	return server.pendingUpdate
}

func (server *Server) SetPendingUpdate(isPending bool) {
	server.pendingUpdate = isPending
}
