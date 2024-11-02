package netdevil

import (
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/I-Am-Dench/goverbuild/models/boot"
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

type Optional[T any] struct {
	Exists bool
	V      T
}

func (o *Optional[T]) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	o.Exists = true
	return d.DecodeElement(&o.V, &start)
}

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
	UGC  Optional[UGC] `xml:"UGC"`
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

func (server *Server) BootConfig(resourceScheme Scheme, useCatalog bool) *boot.Config {
	ugc := UGC{
		Dir:          "3dservices",
		DataCenterId: 150,
	}
	if server.UGC.Exists {
		ugc = server.UGC.V
	}

	patchIP := server.Patcher.Host
	if resourceScheme == File {
		patchIP = "localhost"
		ugc.Host = "localhost"
	}

	patchPort := 80
	if server.Patcher.Port > 0 {
		patchPort = int(server.Patcher.Port)
	}

	return &boot.Config{
		ServerName:       server.Name,
		PatchServerIP:    patchIP,
		AuthServerIP:     server.Game.AuthIP,
		PatchServerPort:  int32(patchPort),
		Logging:          100,
		DataCenterID:     uint32(ugc.DataCenterId),
		CPCode:           0,
		PatchServerDir:   server.Patcher.Dir,
		UGCUse3dServices: server.UGC.Exists,
		UGCServerIP:      ugc.Host,
		UGCServerDir:     ugc.Dir,
		CrashLogURL:      server.Game.CrashLog,
		Locale:           server.Lang,
		ManifestFile:     GameFile,
		UseCatalog:       useCatalog,
	}
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
