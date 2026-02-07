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

func (e Environment) Locale() string {
	return e.UserConfig.Locale
}

func (e Environment) masterIndexUrl(serviceUrl string, resources origin.Resources) string {
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

func (e Environment) GetMasterIndex(ctx context.Context, serviceUrl string, resources origin.Resources) (patcher.MasterIndex, error) {
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

func (e Environment) GetStatusList(ctx context.Context, r origin.Resources, masterIndex patcher.MasterIndex) map[string]*patcher.Status {
	statuses := map[string]*patcher.Status{}

	s, err := masterIndex.GetStatusList(ctx, r)
	if err != nil {
		return statuses
	}

	for _, status := range s {
		statuses[status.Name] = &status
	}

	return statuses
}

func (e Environment) GetServerList(ctx context.Context, r origin.Resources, masterIndex patcher.MasterIndex) ([]patcher.Server, error) {
	universeConfigUrl, err := url.JoinPath(masterIndex.UniverseConfig.URL, "xml", "EnvironmentInfo")
	if err != nil {
		return nil, fmt.Errorf("server list: %w", err)
	}

	reader, err := r.Get(ctx, universeConfigUrl)
	if err != nil {
		return nil, fmt.Errorf("server list: %w", err)
	}
	defer reader.Close()

	universeEnv := UniverseEnvironment{}
	if err := xml.NewDecoder(reader).Decode(&universeEnv); err != nil {
		return nil, fmt.Errorf("server list: %w", err)
	}

	statuses := e.GetStatusList(ctx, r, masterIndex)

	servers := make([]patcher.Server, 0, len(universeEnv.Servers))
	for _, server := range universeEnv.Servers {
		if server.VersionDirType == VersionDirTypePatcherDirVersion {
			server.CdnInfo.PatcherDir = path.Join(server.CdnInfo.PatcherDir, server.Version)
		}

		servers = append(servers, Server{
			UniverseConfig: server,
			UserConfig:     e.UserConfig,
			patcherIniUrl:  universeEnv.PatcherInfo.ConfigUrl,
			status:         statuses[server.Name],
			resources:      r,
		})
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
