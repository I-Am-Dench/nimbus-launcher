package netdevil

import (
	"encoding/xml"
	"fmt"
	"strings"
)

type MasterIndex struct {
	Authentication string `xml:"Authentication"`
	Config         struct {
		XMLName xml.Name `xml:"Config"`
		Type    string   `xml:"type,attr"`
		URL     string   `xml:",chardata"`
	}
	Status string `xml:"Status"`
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
	Game struct {
		AuthIP   string `xml:"AuthIP"`
		CrashLog string `xml:"CrashLog"`
	} `xml:"Game"`
}

func (server *Server) PatchServerUrl(resourceScheme Scheme) string {
	if resourceScheme == File {
		return "file:///" + server.Patcher.Host
	}

	scheme := "http"
	if server.Patcher.Port == 443 || server.Patcher.Port == 8443 {
		scheme = "https"
	}

	if server.Patcher.Port == 443 || server.Patcher.Port == 80 {
		return fmt.Sprint(scheme, "://", server.Patcher.Host)
	}

	return fmt.Sprint(scheme, "://", server.Patcher.Host, ":", server.Patcher.Port)
}

type ServerList struct {
	XMLName xml.Name  `xml:"ServerList"`
	Servers []*Server `xml:"Server"`
}

func (list *ServerList) FindBest(locale string) (*Server, bool) {
	if len(list.Servers) == 0 {
		return nil, false
	}

	if len(locale) == 0 {
		for _, server := range list.Servers {
			if server.Online {
				return server, true
			}
		}
		return nil, false
	}

	best := (*Server)(nil)
	for _, server := range list.Servers {
		if server.Online {
			if strings.EqualFold(server.Lang, locale) {
				best = server
			} else if server == nil {
				best = server
			}
		}
	}

	return best, true
}
