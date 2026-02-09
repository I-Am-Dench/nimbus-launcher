package nimbus

import (
	"encoding/xml"
	"strconv"
	"strings"

	"github.com/I-Am-Dench/nimbus-launcher/internal/optional"
	"github.com/I-Am-Dench/nimbus-launcher/patcher"
	"github.com/I-Am-Dench/nimbus-launcher/patcher/origin"
)

type UGC struct {
	Host         string `xml:"Host"`
	Dir          string `xml:"Dir"`
	DataCenterId int    `xml:"DataCenterId"`
}

type Server struct {
	Name    string `xml:"name,attr"`
	Lang    string `xml:"lang,attr"`
	Online  bool   `xml:"Online"`
	Version string `xml:"Version"`
	Patcher struct {
		Host string `xml:"Host"`
		Dir  string `xml:"Dir"`
		Port uint16 `xml:"Port"`
	} `xml:"Patcher"`
	UGC  optional.O[UGC] `xml:"UGC"`
	Game struct {
		AuthIP   string `xml:"AuthIP"`
		CrashLog string `xml:"CrashLog"`
	} `xml:"Game"`

	status     *patcher.Status  `xml:"-"`
	resources  origin.Resources `xml:"-"`
	userConfig UserConfig       `xml:"-"`
}

func (s Server) PatcherUrl(resources origin.Resources) string {
	if _, ok := resources.(*origin.FS); ok {
		return s.Patcher.Host
	}

	scheme := "http"
	if s.Patcher.Port == 443 || s.Patcher.Port == 8443 {
		scheme = "https"
	}

	if s.Patcher.Port == 443 || s.Patcher.Port == 80 {
		return scheme + "://" + s.Patcher.Host
	}

	return scheme + "://" + s.Patcher.Host + ":" + strconv.FormatUint(uint64(s.Patcher.Port), 10)
}

func (s Server) Info() patcher.ServerInfo {
	return patcher.ServerInfo{
		Name:   s.Name,
		Lang:   s.Lang,
		AuthIP: s.Game.AuthIP,
	}
}

func (s Server) Status() *patcher.Status {
	return s.status
}

func (s Server) GetPatcher(options patcher.Options) (patcher.Patcher, error) {
	return &Patcher{
		UserConfig: s.userConfig,
		Downloader: Downloader{
			Resources: s.resources,
			Log:       options.Log,
			Root:      options.InstallDirectory,
			TempDir:   VersionsDir,
		},
		serverId: options.ServerId,
		server:   s,
	}, nil
}

type ServerList struct {
	XMLName xml.Name `xml:"ServerList"`
	Servers []Server `xml:"Server"`
}

func (l *ServerList) FindBest(locale string) (Server, bool) {
	if len(l.Servers) == 0 {
		return Server{}, false
	}

	var best *Server
	for _, server := range l.Servers {
		if server.Online {
			if strings.EqualFold(server.Lang, locale) {
				return server, true
			} else if best == nil {
				best = &server
			}
		}
	}

	if best != nil {
		return *best, true
	}
	return Server{}, false
}
