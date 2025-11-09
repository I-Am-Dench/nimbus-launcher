package netdevil

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/url"
	"path"
	"path/filepath"

	"github.com/I-Am-Dench/nimbus-launcher/patcher"
	"github.com/I-Am-Dench/nimbus-launcher/patcher/origin"
)

type UserConfig struct {
	Locale       string `json:"locale" xml:"locale"`
	FullDownload bool   `json:"fullDownload" xml:"-"`
}

type Environment struct {
	Environment string `json:"environment" xml:"environment"`
	UserConfig
}

func (e *Environment) Locale() string {
	return e.UserConfig.Locale
}

func (e *Environment) masterIndexUrl(serviceUrl string, resources origin.Resources) string {
	if len(e.Environment) == 0 {
		return serviceUrl
	}

	switch resources.(type) {
	case *origin.FS:
		return filepath.Join(serviceUrl, e.Environment+".xml")
	case *origin.Http:
		return serviceUrl + "?environment=" + e.Environment
	default:
		return serviceUrl
	}
}

func (e *Environment) GetMasterIndex(ctx context.Context, serviceUrl string, resources origin.Resources) (patcher.MasterIndex, error) {
	reader, err := resources.Get(ctx, e.masterIndexUrl(serviceUrl, resources))
	if err != nil {
		return patcher.MasterIndex{}, err
	}
	defer reader.Close()

	masterIndex := patcher.MasterIndex{}
	if err := xml.NewDecoder(reader).Decode(&masterIndex); err != nil {
		return patcher.MasterIndex{}, err
	}

	return masterIndex, nil
}

func (e Environment) GetServerList(ctx context.Context, r origin.Resources, masterIndex patcher.MasterIndex) ([]patcher.Server, error) {
	reader, err := r.Get(ctx, masterIndex.UniverseConfig.URL)
	if err != nil {
		return nil, fmt.Errorf("server list: %w", err)
	}
	defer reader.Close()

	serverList := ServerList{}
	if err := xml.NewDecoder(reader).Decode(&serverList); err != nil {
		return nil, fmt.Errorf("server list: %v", err)
	}

	servers := make([]patcher.Server, 0, len(serverList.Servers))
	for _, server := range serverList.Servers {
		resources := r

		switch v := resources.(type) {
		case *origin.FS:
			resources = origin.WithRoot(v, path.Join(server.Patcher.Host, server.Patcher.Dir))
		case *origin.Http, *origin.HttpWithAuth:
			u, err := url.JoinPath(server.PatcherUrl(v), server.Patcher.Dir)
			if err != nil {
				return nil, fmt.Errorf("nd-nimbus: %v", err)
			}
			resources = origin.WithUrl(resources, u)
		}

		server.resources = resources
		server.userConfig = e.UserConfig
		servers = append(servers, server)
	}

	return servers, nil
}

func cancelled(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}
