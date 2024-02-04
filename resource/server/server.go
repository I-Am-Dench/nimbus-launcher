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

	hasPatchesList bool          `json:"-"`
	patchesList    patch.Summary `json:"-"`
	pendingUpdate  bool          `json:"-"`
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

// Returns the path in the format: {server.downloadDir}/{server.ID}
func (server *Server) DownloadDir() string {
	return filepath.Join(server.downloadDir, server.ID)
}

// Returns the path in the format: {server.settingsDir}/servers/{server.Boot}
func (server *Server) BootPath() string {
	return filepath.Join(server.settingsDir, "servers", server.Boot)
}

// Marshals the contents of server.Config and writes it to the path specified by
// server.BootConfig().
//
// If len(server.Boot) == 0 when this method is called, server.Boot will be set to a
// string with the format: "boot_{server.ID}.cfg".
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

// Reads the file located by server.BootPath() and unmarshals it into server.Config.
//
// If len(server.Boot) == 0, this function return immediately with no error
//
// This method can be optionally called with up to 2 parameters. The first optional parameter
// will always update the internal settingsDir field. The second option parameter will always
// update the internal downloadDir field.
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

// Returns a string formatted as: server.PatchProtocol://PatchServerIP:PatchServerPort
func (server *Server) PatchServerHost() string {
	return fmt.Sprint(server.PatchProtocol, "://", server.Config.PatchServerIP, ":", server.Config.PatchServerPort)
}

// Calls url.JoinPath to format a valid URL in the form: server.PatchServerHost()/PatchServerDir/{elem...}
//
// For example:
//
//	// server.PatchServerHost() returns "http://127.0.0.1:10000"
//	// PatchServerDir = "patches"
//
//	server.PatchServerUrl("some", "content", "here") // returns "http://127.0.0.1:10000/patches/some/content/here"
func (server *Server) PatchServerUrl(elem ...string) (string, error) {
	path := []string{server.Config.PatchServerDir}
	return url.JoinPath(server.PatchServerHost(), append(path, elem...)...)
}

// Fetches the patch.json for the specified version.
//
// This method first checks for a file called patch.json located in the directory
// "server.DownloadDir()/{version}". If the file does not exist, the patch.json is
// request from the remote by calling server.RemoteGet(version, "patch.json").
//
// If server.RemoteGet returns an error, patch.ErrPatchesUnavailable is returned.
//
// If server.RemoteGet returns a status code of 401, patch.ErrPatchesUnauthorized is returned.
//
// If server.RemoteGet returns any other status code >= 400, patch.ErrPatchesUnavailable is returned.
//
// If the contents of the patch.json are formatted correctly (calling json.Marshal on the data does not return an error),
// the data is saved in the file "server.DownloadDir()/{version}/patch.json".
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

	response, err := server.RemoteGet(version, "patch.json")
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

// Returns an *http.Response after sending a request to the url created by server.PatchServerUrl(elem...).
//
// If the len(server.PatchToken) > 0, the TPP-Token header is added to the request with the value of server.PatchToken.
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

// Sends an HTTP request to at the URL formatted with server.PatchServerUrl().
//
// If the request fails, patch.ErrPatchesUnavailable is returned.
//
// If the response returns a status code of 503, patch.ErrPatchesUnsupported is returned.
func (server *Server) GetPatchesSummary() (patch.Summary, error) {
	response, err := server.RemoteGet("summary.json")
	if err != nil {
		return patch.Summary{}, patch.ErrPatchesUnavailable
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusServiceUnavailable {
		return patch.Summary{}, patch.ErrPatchesUnsupported
	}

	if response.StatusCode >= 400 {
		return patch.Summary{}, fmt.Errorf("invalid response status code from server: %d", response.StatusCode)
	}

	data, err := io.ReadAll(response.Body)
	if err != nil {
		return patch.Summary{}, fmt.Errorf("cannot read body of patch server response: %w", err)
	}

	patches := patch.Summary{}
	err = json.Unmarshal(data, &patches)
	if err != nil {
		return patch.Summary{}, fmt.Errorf("malformed response body from server: %w", err)
	}

	return patches, nil
}

// Sets server.Config to boot and then calls server.SaveConfig() returning the error.
func (server *Server) SetBootConfig(boot *ldf.BootConfig) error {
	server.Config = boot
	return server.SaveConfig()
}

func (server *Server) SetPatchProtocol(protocol string) {
	server.PatchProtocol = protocol
}

func (server *Server) ToXML() XML {
	data, _ := ldf.Marshal(server.Config)
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

// Returns the PatchesSummary saved from a call to server.SetPatchesSummary.
//
// If server.SetPatchesSummary has not yet been called, this method returns false.
func (server *Server) PatchesSummary() (patch.Summary, bool) {
	return server.patchesList, server.hasPatchesList
}

// Internally stores the summary
func (server *Server) SetPatchesSummary(summary patch.Summary) {
	server.patchesList = summary
	server.hasPatchesList = true
}

func (server *Server) PendingUpdate() bool {
	return server.pendingUpdate
}

func (server *Server) SetPendingUpdate(isPending bool) {
	server.pendingUpdate = isPending
}
