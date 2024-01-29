package server

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/I-Am-Dench/lu-launcher/ldf"
	"github.com/I-Am-Dench/lu-launcher/resource/patch"
)

const (
	HEADER_PATCH_TOKEN = "TPP-Token"
)

type PatchesSummary struct {
	CurrentVersion   string   `json:"currentVersion"`
	PreviousVersions []string `json:"previousVersions"`
}

var _ patch.Server = &Server{}

type Server struct {
	settingsDir string `json:"-"`
	downloadDir string `json:"-"`

	ID            string `json:"id"`
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

func New(config Config) *Server {
	return &Server{
		settingsDir: config.SettingsDir,
		downloadDir: config.DownloadDir,

		ID:            fmt.Sprint(time.Now().Unix()),
		Name:          config.Name,
		PatchToken:    config.PatchToken,
		PatchProtocol: config.PatchProtocol,
		Config:        config.Config,
	}
}

func (server *Server) Id() string {
	return server.ID
}

func (server *Server) DownloadDir() string {
	return filepath.Join(server.downloadDir, server.ID)
}

func (server *Server) BootPath() string {
	return filepath.Join(server.settingsDir, "servers", server.Boot)
}

func (server *Server) SaveConfig() error {
	if len(server.Boot) == 0 {
		server.Boot = fmt.Sprintf("boot_%s.cfg", server.ID)
	}

	data, err := ldf.MarshalLines(server.Config)
	if err != nil {
		return fmt.Errorf("cannot marshal boot.cfg: %w", err)
	}

	err = os.WriteFile(server.BootPath(), data, 0755)
	if err != nil {
		return fmt.Errorf("cannot save boot.cfg: %w", err)
	}

	return nil
}

func (server *Server) LoadConfig(dir ...string) error {
	if len(dir) > 0 {
		server.settingsDir = dir[0]
	}

	if len(dir) > 1 {
		server.downloadDir = dir[1]
	}

	server.Config = &ldf.BootConfig{}

	if len(server.Boot) == 0 {
		return nil
	}

	data, err := os.ReadFile(server.BootPath())
	if err != nil {
		return fmt.Errorf("cannot load boot.cfg: %w", err)
	}

	err = ldf.Unmarshal(data, server.Config)
	if err != nil {
		return fmt.Errorf("cannot unmarshal boot.cfg: %w", err)
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

func (server *Server) GetPatch(version string) (patch.Patch, error) {
	patchDirectory := filepath.Join(server.DownloadDir(), version)
	path := filepath.Join(patchDirectory, "patch.json")

	data, err := os.ReadFile(path)
	if err == nil {
		patch := patch.NewTpp(version)

		err = json.Unmarshal(data, &patch)
		if err != nil {
			return nil, fmt.Errorf("cannot unmarshal \"%s\": %w", path, err)
		}

		return patch, nil
	}

	response, err := server.RemoteGet(version)
	if err != nil {
		return nil, patch.ErrPatchesUnavailable
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusUnauthorized {
		return nil, patch.ErrPatchesUnauthorized
	}

	if response.StatusCode >= 400 {
		return nil, patch.ErrPatchesUnavailable
	}

	data, err = io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("cannot read body of patch version response: %w", err)
	}

	patch := patch.NewTpp(version)
	err = json.Unmarshal(data, &patch)
	if err != nil {
		return nil, fmt.Errorf("malformed response body from patch version: %w", err)
	}

	os.MkdirAll(patchDirectory, 0755)

	err = os.WriteFile(path, data, 0755)
	if err != nil {
		log.Printf("Could not save patch.json: %v", err)
	}

	return patch, err
}

func (server *Server) RemoteGet(elem ...string) (*http.Response, error) {
	url, err := server.PatchServerUrl(elem...)
	if err != nil {
		return nil, fmt.Errorf("could not create patch url: %w", err)
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

func (server *Server) GetPatchesSummary() (PatchesSummary, error) {
	response, err := server.RemoteGet()
	if err != nil {
		return PatchesSummary{}, patch.ErrPatchesUnavailable
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusServiceUnavailable {
		return PatchesSummary{}, patch.ErrPatchesUnsupported
	}

	if response.StatusCode >= 400 {
		return PatchesSummary{}, fmt.Errorf("invalid response status code from server: %d", response.StatusCode)
	}

	data, err := io.ReadAll(response.Body)
	if err != nil {
		return PatchesSummary{}, fmt.Errorf("cannot read boyd of patch server response: %w", err)
	}

	patches := PatchesSummary{}
	err = json.Unmarshal(data, &patches)
	if err != nil {
		return PatchesSummary{}, fmt.Errorf("malformed response body from server: %w", err)
	}

	return patches, nil
}

func (server *Server) SetBootConfig(boot *ldf.BootConfig) error {
	server.Config = boot
	return server.SaveConfig()
}

func (server *Server) SetPatchProtocol(protocol string) {
	server.PatchProtocol = protocol
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
