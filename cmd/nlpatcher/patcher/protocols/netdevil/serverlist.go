package netdevil

import (
	"encoding/xml"
	"fmt"
	"strings"
)

type Server struct {
	Name           string `xml:"name,attr"`
	Lang           string `xml:"lang,attr"`
	Online         bool   `xml:"Online"`
	Version        string `xml:"Version"`
	PatchIP        string `xml:"PatchIP"`
	PatchPort      uint16 `xml:"PatchPort"`
	PatchServerDir string `xml:"PatchServerDir"`
	Config         string `xml:"Config"`
}

func (server *Server) PatchServerUrl(local bool) string {
	if local {
		return server.PatchIP
	}

	scheme := "http"
	if server.PatchPort == 443 || server.PatchPort == 8443 {
		scheme = "https"
	}

	if server.PatchPort == 443 || server.PatchPort == 80 {
		return fmt.Sprint(scheme, "://", server.PatchIP)
	}

	return fmt.Sprint(scheme, "://", server.PatchIP, ":", server.PatchPort)
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
