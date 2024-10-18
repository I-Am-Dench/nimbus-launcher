package netdevil

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"

	"github.com/I-Am-Dench/nimbus-launcher/cmd/nlpatcher/patcher"
	"github.com/I-Am-Dench/nimbus-launcher/cmd/nlpatcher/patcher/protocols/netdevil/resources"
)

type Scheme int

const (
	File = Scheme(iota)
	Http
)

type NetDevilPatcher struct {
	patcher.Config

	client *http.Client `json:"-"`
	scheme Scheme       `json:"-"`

	GetResource resources.Func `json:"-"`

	ServiceUrl  string `json:"serviceUrl"`
	Environment string `json:"environment"`

	AuthUrl string `json:"authUrl"`

	Patch patchConfig `json:"patch"`
}

func (patcher *NetDevilPatcher) Authenticate() (bool, error) {
	if patcher.scheme == File || patcher.CredentialsFunc == nil {
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

func (patcher *NetDevilPatcher) FetchMasterIndex() (MasterIndex, error) {
	var masterIndexUrl string
	if patcher.scheme == File {
		masterIndexUrl = filepath.Join(patcher.ServiceUrl, patcher.Environment+".xml")
	} else {
		masterIndexUrl = patcher.ServiceUrl + "?environment=" + patcher.Environment
	}

	resource, err := patcher.GetResource(masterIndexUrl)
	if err != nil {
		return MasterIndex{}, fmt.Errorf("fetch master index: %w", err)
	}
	defer resource.Close()

	masterIndex := MasterIndex{}
	if err := xml.NewDecoder(resource).Decode(&masterIndex); err != nil {
		return MasterIndex{}, fmt.Errorf("fetch master index: %w", err)
	}

	return masterIndex, nil
}

func (patcher *NetDevilPatcher) FetchConfig(url string) (ServerList, error) {
	resource, err := patcher.GetResource(url)
	if err != nil {
		return ServerList{}, fmt.Errorf("fetch config: %w", err)
	}
	defer resource.Close()

	serverList := ServerList{}
	if err := xml.NewDecoder(resource).Decode(&serverList); err != nil {
		return ServerList{}, fmt.Errorf("fetch config: %w", err)
	}

	return serverList, nil
}

func (patcher *NetDevilPatcher) patchResourceFunc(server *Server) (resources.Func, error) {
	uri, err := url.ParseRequestURI(server.PatchServerUrl(patcher.scheme))
	if err != nil {
		return nil, fmt.Errorf("invalid uri: %w", err)
	}

	uri = uri.JoinPath(server.Patcher.Dir)

	resourceFunc, scheme, err := patcher.NewResourcesFunc(uri)
	if err != nil {
		return nil, err
	}

	if scheme == File {
		resourceFunc = resources.WithFileBase(resourceFunc, resources.FileSchemePath(uri.Path))
	} else if scheme == Http {
		resourceFunc = resources.WithUrlBase(resourceFunc, uri.String())
	}

	return resourceFunc, nil
}

func (patcher *NetDevilPatcher) GetPatch(options patcher.PatchOptions) (patcher.Patch, error) {
	masterIndex, err := patcher.FetchMasterIndex()
	if err != nil {
		return nil, fmt.Errorf("netdevil: %w", err)
	}

	if masterIndex.Config.Type != "nd-nimbus" {
		return nil, fmt.Errorf("netdevil: %w", err)
	}

	servers, err := patcher.FetchConfig(masterIndex.Config.URL)
	if err != nil {
		return nil, fmt.Errorf("netdevil: %w", err)
	}

	patcher.Patch.PatchOptions = options

	// config := patchConfig{
	// 	PatchOptions: options,
	// }
	// json.Unmarshal(patcher.patchConfig, &config)

	server, ok := servers.FindBest(patcher.Patch.Locale)
	if !ok {
		return nil, errors.New("netdevil: no servers available")
	}

	patchResourceFunc, err := patcher.patchResourceFunc(server)
	if err != nil {
		return nil, fmt.Errorf("netdevil: %w", err)
	}

	patcher.Patch.Server = server
	patcher.Patch.GetResource = patchResourceFunc

	return newPatch(patcher.Patch)
}

func (p *NetDevilPatcher) NewResourcesFunc(uri *url.URL) (resources.Func, Scheme, error) {
	switch uri.Scheme {
	default:
		return nil, 0, fmt.Errorf("unknown service scheme: %s", uri.Scheme)
	case "file":
		if p.ForceRemoteResources {
			return nil, 0, errors.New("local file paths are disallowed")
		}

		return resources.File, File, nil
	case "http", "https":
		return resources.Http(p.client), Http, nil
	}
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

	resourceFunc, scheme, err := p.NewResourcesFunc(uri)
	if err != nil {
		return nil, fmt.Errorf("netdevil: %w", err)
	}

	p.GetResource = resourceFunc
	p.scheme = scheme

	if scheme == File {
		p.ServiceUrl = resources.FileSchemePath(uri.Path)
	}

	return p, nil
}
