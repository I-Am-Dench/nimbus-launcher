package netdevil

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"

	"github.com/I-Am-Dench/nimbus-launcher/cmd/nlpatcher/patcher"
)

type NetDevilPatcher struct {
	patcher.Config

	client  *http.Client `json:"-"`
	isLocal bool         `json:"-"`

	resourceFunc ResourceFunc `json:"-"`

	ServiceUrl  string `json:"serviceUrl"`
	Environment string `json:"environment"`

	AuthUrl string `json:"authUrl"`
}

func (patcher *NetDevilPatcher) Authenticate() (bool, error) {
	if patcher.isLocal || patcher.CredentialsFunc == nil {
		return true, nil
	}

	username, password, err := patcher.CredentialsFunc()
	if err != nil {
		return false, fmt.Errorf("authenticate: %w", err)
	}

	request, err := http.NewRequest(http.MethodPost, patcher.AuthUrl, nil)
	if err != nil {
		return false, fmt.Errorf("authenticate: %w", err)
	}

	request.SetBasicAuth(username, string(password))

	response, err := patcher.client.Do(request)
	if err != nil {
		return false, fmt.Errorf("authenticate: %w", err)
	}

	if response.StatusCode >= 200 && response.StatusCode < 300 {
		return true, nil
	}

	switch response.StatusCode {
	case http.StatusBadRequest, http.StatusUnauthorized:
		return false, nil
	default:
		return false, fmt.Errorf("authenticate: unhandled status: %s", response.Status)
	}
}

func (patcher *NetDevilPatcher) GetPatch(options patcher.PatchOptions) (patcher.Patch, bool, error) {
	var serviceUri string
	if patcher.isLocal {
		serviceUri = filepath.Join(patcher.ServiceUrl, patcher.Environment+".xml")
	} else {
		serviceUri = fmt.Sprint(patcher.ServiceUrl, "?environment=", patcher.Environment)
	}

	resource, err := patcher.resourceFunc(serviceUri)
	if err != nil {
		return nil, false, fmt.Errorf("netdevil: environment: %w", err)
	}
	defer resource.Close()

	servers := ServerList{}
	if err := xml.NewDecoder(resource).Decode(&servers); err != nil {
		return nil, false, fmt.Errorf("netdevil: %w", err)
	}

	config := patchConfig{
		PatchOptions: options,
		ResourceFunc: patcher.resourceFunc,
	}
	json.Unmarshal(options.Config, &config)

	server, ok := servers.FindBest(config.Locale)
	if !ok {
		return nil, false, errors.New("netdevil: no servers available")
	}

	config.Server = server
	config.PatchUrl = path.Join(server.PatchServerUrl(patcher.isLocal), server.PatchServerDir)

	patch, err := newPatch(config)
	if err != nil {
		return nil, false, fmt.Errorf("netdevil: %w", err)
	}

	currentVersion, err := patch.Manifest("version.txt")
	if err != nil {
		return nil, false, fmt.Errorf("netdevil: %w", err)
	}

	return patch, currentVersion.Version != patch.ManifestFile.Version, nil
}

func (patcher *NetDevilPatcher) fetchFileResource(uri string) (io.ReadCloser, error) {
	file, err := os.Open(filepath.FromSlash(uri))
	if err != nil {
		return nil, fmt.Errorf("fetch file resource: %w", err)
	}

	return file, nil
}

func (patcher *NetDevilPatcher) fetchHttpResource(uri string) (io.ReadCloser, error) {
	request, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		return nil, fmt.Errorf("fetch http resource: %w", err)
	}

	response, err := patcher.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("fetch http resource: %w", err)
	}

	if response.StatusCode >= 200 && response.StatusCode < 300 {
		return response.Body, nil
	}

	// Enables the client to reuse the connection when using Keep-Alive
	io.Copy(io.Discard, response.Body)
	response.Body.Close()

	return nil, fmt.Errorf("fetch http resource: unhandled status: %s", response.Status)
}

func New(r io.Reader, config patcher.Config) (patcher.Patcher, error) {
	p := &NetDevilPatcher{
		Config: config,

		client: &http.Client{
			Jar:       config.CookieJar,
			Transport: http.DefaultTransport,
		},
	}

	if err := json.NewDecoder(r).Decode(&p); err != nil {
		return nil, fmt.Errorf("netdevil: %w", err)
	}

	uri, err := url.ParseRequestURI(p.ServiceUrl)
	if err != nil {
		return nil, fmt.Errorf("netdevil: invalid uri: %w", err)
	}

	switch uri.Scheme {
	default:
		return nil, fmt.Errorf("netdevil: unknown service scheme: %s", uri.Scheme)
	case "file":
		if config.ForceRemoteResources {
			return nil, fmt.Errorf("netdevil: local file paths are disallowed")
		}

		p.ServiceUrl = patcher.FileSchemeToPath(uri.Path)
		p.isLocal = true

		p.resourceFunc = p.fetchFileResource
	case "http", "https":
		p.resourceFunc = p.fetchHttpResource
	}

	return p, nil
}
