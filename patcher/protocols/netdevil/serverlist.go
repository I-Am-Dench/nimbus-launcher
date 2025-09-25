package netdevil

import (
	"encoding/xml"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/I-Am-Dench/goverbuild/models/boot"
	"github.com/I-Am-Dench/nimbus-launcher/patcher/origin"
)

type Optional[T any] struct {
	Exists bool
	Value  T
}

func (o *Optional[T]) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	o.Exists = true
	return d.DecodeElement(&o.Value, &start)
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
	}
	UGC  Optional[UGC] `xml:"UGC"`
	Game struct {
		AuthIP   string `xml:"AuthIP"`
		CrashLog string `xml:"CrashLog"`
	} `xml:"Game"`
}

func (s *Server) PatcherUrl(resources origin.Resources) string {
	if _, ok := resources.(*origin.FS); ok {
		return filepath.Join(s.Patcher.Host, s.Patcher.Dir)
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

func (s *Server) Boot(locale string, useCatalog bool) *boot.Config {
	ugc := UGC{
		Host:         "localhost",
		Dir:          "3dservices",
		DataCenterId: 150,
	}
	if s.UGC.Exists {
		ugc = s.UGC.Value
	}

	patchPort := 80
	if s.Patcher.Port > 0 {
		patchPort = int(s.Patcher.Port)
	}

	return &boot.Config{
		ServerName:       s.Name,
		PatchServerIP:    s.Patcher.Host,
		AuthServerIP:     s.Game.AuthIP,
		PatchServerPort:  int32(patchPort),
		Logging:          100,
		DataCenterID:     uint32(ugc.DataCenterId),
		PatchServerDir:   s.Patcher.Dir,
		UGCUse3dServices: s.UGC.Exists,
		UGCServerIP:      ugc.Host,
		UGCServerDir:     ugc.Dir,
		CrashLogURL:      s.Game.CrashLog,
		Locale:           locale,
		ManifestFile:     GameFile,
		UseCatalog:       useCatalog,
	}
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
