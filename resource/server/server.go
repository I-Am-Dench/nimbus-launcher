package server

import (
	"encoding/xml"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/I-Am-Dench/lu-launcher/ldf"
)

const (
	HEADER_PATCH_TOKEN = "TPP-Token"
)

type PatchesSummary struct {
	CurrentVersion   string   `json:"currentVersion"`
	PreviousVersions []string `json:"previousVersions"`
}

type Server struct {
	dir string `json:"-"`

	Id            string `json:"id"`
	Name          string `json:"name"`
	Boot          string `json:"boot"`
	PatchToken    string `json:"patchToken"`
	PatchProtocol string `json:"patchProtocol"`
	CurrentPatch  string `json:"currentPatch"`

	Config *ldf.BootConfig `json:"-"`

	hasPatchesList bool           `json:"-"`
	patchesList    PatchesSummary `json:"-"`
	pendingUpdate  bool           `json:"-"`
}

func New(dir string, name, patchToken, patchProtocol string, config *ldf.BootConfig) *Server {
	return &Server{
		dir: dir,

		Id:            fmt.Sprint(time.Now().Unix()),
		Name:          name,
		PatchToken:    patchToken,
		PatchProtocol: patchProtocol,
		Config:        config,
	}
}

func Create(name, patchToken, patchProtocol string, config *ldf.BootConfig) (*Server, error) {
	server := New("settings", name, patchToken, patchProtocol, config)
	return server, server.SaveConfig()
}

func (server *Server) BootPath() string {
	return filepath.Join(server.dir, "servers", server.Boot)
}

func (server *Server) SaveConfig() error {
	if len(server.Boot) == 0 {
		server.Boot = fmt.Sprintf("boot_%s.cfg", server.Id)
	}

	data, err := ldf.MarshalLines(server.Config)
	if err != nil {
		return fmt.Errorf("cannot marshal boot.cfg: %v", err)
	}

	err = os.WriteFile(server.BootPath(), data, 0755)
	if err != nil {
		return fmt.Errorf("cannot save boot.cfg: %v", err)
	}

	return nil
}

func (server *Server) LoadConfig() error {
	server.Config = &ldf.BootConfig{}

	if len(server.Boot) == 0 {
		return nil
	}

	data, err := os.ReadFile(server.BootPath())
	if err != nil {
		return fmt.Errorf("cannot load boot.cfg: %v", err)
	}

	err = ldf.Unmarshal(data, server.Config)
	if err != nil {
		return fmt.Errorf("cannot unmarshal boot.cfg: %v", err)
	}

	return nil
}

func (server *Server) DeleteConfig() error {
	return os.Remove(server.BootPath())
}

func (server *Server) PatchServerHost() string {
	return fmt.Sprint(server.PatchProtocol, "://", server.Config.PatchServerIP, ":", server.Config.PatchServerPort)
}

func (server *Server) PatchServerUrl(elem ...string) (string, error) {
	path := []string{server.Config.PatchServerDir}
	return url.JoinPath(server.PatchServerHost(), append(path, elem...)...)
}

func (server *Server) RemoteGet(elem ...string) (*http.Response, error) {
	url, err := server.PatchServerUrl(elem...)
	if err != nil {
		return nil, fmt.Errorf("could not create patch url: %v", err)
	}

	log.Printf("Patch server request: %s", url)
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

func (server *Server) ToXML() XML {
	data, _ := ldf.MarshalLines(server.Config)
	return XML{
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

func (server *Server) PatchesSummary() (PatchesSummary, bool) {
	return server.patchesList, server.hasPatchesList
}

func (server *Server) SetPatchesSummary(summary PatchesSummary) {
	server.patchesList = summary
	server.hasPatchesList = true
}

func (server *Server) PendingUpdate() bool {
	return server.pendingUpdate
}

func (server *Server) SetPendingUpdate(isPending bool) {
	server.pendingUpdate = isPending
}
