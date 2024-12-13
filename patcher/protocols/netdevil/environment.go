package netdevil

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"net/url"
	"path"
	"path/filepath"

	"github.com/I-Am-Dench/nimbus-launcher/patcher"
	"github.com/I-Am-Dench/nimbus-launcher/patcher/resources"
)

type UserConfig struct {
	Locale       string `json:"locale"`
	FullDownload bool   `json:"fullDownload"`
}

type Config struct {
	Environment string `json:"environment"`
	UserConfig
}

func (config *Config) FormatMasterIndexUrl(serviceUrl string, scheme patcher.Scheme) string {
	if scheme == resources.FileScheme {
		return filepath.Join(serviceUrl, config.Environment+".xml")
	} else {
		return serviceUrl + "?environment=" + config.Environment
	}
}

func (config *Config) getServerList(configUrl string, opt patcher.Options) (ServerList, error) {
	resource, err := opt.Resources.Get(configUrl)
	if err != nil {
		return ServerList{}, fmt.Errorf("server list: %w", err)
	}
	defer resource.Close()

	serverList := ServerList{}
	if err := xml.NewDecoder(resource).Decode(&serverList); err != nil {
		return ServerList{}, fmt.Errorf("server list: %w", err)
	}

	return serverList, nil
}

func (config *Config) NewPatcher(ctx context.Context, masterIndex patcher.MasterIndex, opt patcher.Options) (patcher.Patcher, error) {
	if masterIndex.Config.Type != "nd-nimbus" {
		return nil, fmt.Errorf("netdevil: mismatched config type: %s", masterIndex.Config.Type)
	}

	serverList, err := config.getServerList(masterIndex.Config.URL, opt)
	if err != nil {
		return nil, fmt.Errorf("netdevil: %w", err)
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	server, ok := serverList.FindBest(config.Locale)
	if !ok {
		return nil, errors.New("netdevil: no servers available")
	}

	if opt.Resources.Scheme() == resources.FileScheme {
		opt.Resources = resources.WithRoot(opt.Resources, path.Join(server.Patcher.Host, server.Patcher.Dir))
	} else {
		u, err := url.JoinPath(server.PatcherUrl(opt.Resources.Scheme()), server.Patcher.Dir)
		if err != nil {
			return nil, fmt.Errorf("netdevil: %w", err)
		}
		opt.Resources = resources.WithUrl(opt.Resources, u)
	}

	return &Patcher{
		UserConfig: config.UserConfig,
		Options:    opt,

		Server: server,
	}, nil
}
