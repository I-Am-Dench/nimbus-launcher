package nimbus

import (
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"

	"github.com/I-Am-Dench/goverbuild/archive/manifest"
	"github.com/I-Am-Dench/goverbuild/encoding/ldf"
	"github.com/I-Am-Dench/nimbus-launcher/internal/optional"
	"github.com/I-Am-Dench/nimbus-launcher/patcher"
	"github.com/I-Am-Dench/nimbus-launcher/patcher/origin"
	int_netdevil "github.com/I-Am-Dench/nimbus-launcher/patcher/protocols/internal/netdevil"
	"github.com/I-Am-Dench/nimbus-launcher/patcher/protocols/netdevil"
)

const ManifestVersion = 10000

type UGC struct {
	Host         string `xml:"Host"`
	Dir          string `xml:"Dir"`
	DataCenterId int    `xml:"DataCenterId"`
}

type LdfEntries []ldf.Entry

func (l *LdfEntries) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	type KeyValue struct {
		Id    string `xml:"id,attr"`
		Value []byte `xml:",chardata"`
	}

	appendEntries := func(entries *[]ldf.Entry, kvs []KeyValue, valueType ldf.ValueType) error {
		for _, kv := range kvs {
			entry, err := ldf.Token{
				Key:   kv.Id,
				Type:  valueType,
				Value: kv.Value,
			}.Entry()
			if err != nil {
				return err
			}
			*entries = append(*entries, entry)
		}
		return nil
	}

	keys := struct {
		Strings  []KeyValue `xml:"string"`
		Utf8s    []KeyValue `xml:"utf8"`
		Int32s   []KeyValue `xml:"int32"`
		Uint32s  []KeyValue `xml:"uint32"`
		Int64s   []KeyValue `xml:"int64"`
		Uint64s  []KeyValue `xml:"uint64"`
		Float32s []KeyValue `xml:"float32"`
		Float64s []KeyValue `xml:"float64"`
		Bools    []KeyValue `xml:"bool"`
	}{}
	if err := d.DecodeElement(&keys, &start); err != nil {
		return err
	}

	var entries []ldf.Entry
	if err := appendEntries(&entries, keys.Strings, ldf.ValueTypeString); err != nil {
		return err
	}
	if err := appendEntries(&entries, keys.Utf8s, ldf.ValueTypeUtf8); err != nil {
		return err
	}
	if err := appendEntries(&entries, keys.Int32s, ldf.ValueTypeI32); err != nil {
		return err
	}
	if err := appendEntries(&entries, keys.Uint32s, ldf.ValueTypeU32); err != nil {
		return err
	}
	if err := appendEntries(&entries, keys.Int64s, ldf.ValueTypeI64); err != nil {
		return err
	}
	if err := appendEntries(&entries, keys.Uint64s, ldf.ValueTypeU64); err != nil {
		return err
	}
	if err := appendEntries(&entries, keys.Float32s, ldf.ValueTypeFloat); err != nil {
		return err
	}
	if err := appendEntries(&entries, keys.Float64s, ldf.ValueTypeDouble); err != nil {
		return err
	}
	if err := appendEntries(&entries, keys.Bools, ldf.ValueTypeBool); err != nil {
		return err
	}
	*l = entries

	return nil
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
		AuthIP   string     `xml:"AuthIP"`
		CrashLog string     `xml:"CrashLog"`
		Config   LdfEntries `xml:"Config"`
	} `xml:"Game"`

	status    *patcher.Status
	resources origin.Resources
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

func verifyManifest(manifest *manifest.Manifest) error {
	if manifest.Version != ManifestVersion && manifest.Version != netdevil.ManifestVersion {
		return fmt.Errorf("incompatible manifest version: expected either %d or %d but got %d", ManifestVersion, netdevil.ManifestVersion, manifest.Version)
	}
	return nil
}

func (s Server) GetPatcher(options patcher.Options) (patcher.Patcher, error) {
	config := int_netdevil.Config{
		Locale:                 options.Locale,
		DownloadType:           options.DownloadType,
		ServerId:               options.ServerId,
		Version:                s.Version,
		VersionDirType:         int_netdevil.VersionDirTypeVersionHotfixOnly,
		VerifyManifestFunc:     verifyManifest,
		AllowNoMinimalManifest: true,
	}

	downloader := int_netdevil.Downloader{
		Resources: s.resources,
		Log:       options.Log,
		Root:      options.InstallDirectory,
		TempDir:   patcher.VersionsDir,
	}

	return &Patcher{
		patcher: int_netdevil.NewPatcher(config, downloader),
		server:  s,
	}, nil
}

type ServerList struct {
	XMLName xml.Name `xml:"ServerList"`
	Servers []Server `xml:"Server"`
}

func (l ServerList) FindBest(locale string) (Server, bool) {
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
