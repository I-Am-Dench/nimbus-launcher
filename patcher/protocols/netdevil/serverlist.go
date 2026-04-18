package netdevil

import (
	"encoding/xml"
	"fmt"
	"net/url"
	"path"
	"path/filepath"

	"github.com/I-Am-Dench/goverbuild/archive/manifest"
	"github.com/I-Am-Dench/nimbus-launcher/patcher"
	"github.com/I-Am-Dench/nimbus-launcher/patcher/origin"
	int_netdevil "github.com/I-Am-Dench/nimbus-launcher/patcher/protocols/internal/netdevil"
)

const ManifestVersion = 82

type Server struct {
	UniverseConfig

	patcherIniUrl string
	status        *patcher.Status
	resources     origin.Resources
}

func (s Server) Info() patcher.ServerInfo {
	return patcher.ServerInfo{
		Name:   s.Name,
		Lang:   s.Language,
		AuthIP: s.AuthenticationIP,
	}
}

func (s Server) Status() *patcher.Status {
	return s.status
}

func verifyManifest(manifest *manifest.Manifest) error {
	if manifest.Version != ManifestVersion {
		return fmt.Errorf("incompatible manifest version: expected %d but got %d", ManifestVersion, manifest.Version)
	}
	return nil
}

func (s Server) GetPatcher(options patcher.Options) (patcher.Patcher, error) {
	resources := s.resources

	switch v := resources.(type) {
	case *origin.FS:
		resources = origin.WithRoot(v, filepath.Join(s.CdnInfo.PatcherUrl, s.CdnInfo.PatcherDir))
	case *origin.Http, *origin.HttpWithAuth:
		u, err := url.JoinPath(s.PatcherUrl(v), s.CdnInfo.PatcherDir)
		if err != nil {
			return nil, fmt.Errorf("netdevil: %v", err)
		}
		resources = origin.WithUrl(resources, u)
	}

	config := int_netdevil.Config{
		Locale:             options.Locale,
		DownloadType:       options.DownloadType,
		ServerId:           options.ServerId,
		Version:            s.Version,
		VersionDirType:     s.VersionDirType,
		VerifyManifestFunc: verifyManifest,
	}

	downloader := int_netdevil.Downloader{
		Resources: resources,
		Log:       options.Log,
		Root:      options.InstallDirectory,
		TempDir:   patcher.VersionsDir,
	}

	return &Patcher{
		patcher: int_netdevil.NewPatcher(config, downloader),
		server:  s,
	}, nil
}

type VersionDirType = int_netdevil.VersionDirType

type UniverseConfig struct {
	AuthenticationIP string `xml:"AuthenticationIP"`
	CdnInfo          struct {
		PatcherUrl string `xml:"PatcherUrl"`
		PatcherDir string `xml:"PatcherDir"`
		Secure     bool   `xml:"Secure"`
	} `xml:"CdnInfo"`
	UgcCdnInfo struct {
		PatcherUrl string `xml:"PatcherUrl"`
		PatcherDir string `xml:"PatcherDir"`
		Secure     bool   `xml:"Secure"`
	} `xml:"UgcCdnInfo"`
	DataCenterId   int            `xml:"DataCenterID"`
	Language       string         `xml:"Language"`
	Online         bool           `xml:"Online"`
	Name           string         `xml:"Name"`
	LogLevel       int32          `xml:"LogLevel"`
	Use3dServices  bool           `xml:"Use3DServices"`
	UseDB          bool           `xml:"UseDB"`
	Version        string         `xml:"Version"`
	VersionDirType VersionDirType `xml:"VersionDirType"`
}

func (c UniverseConfig) PatcherUrl(resources origin.Resources) string {
	if _, ok := resources.(*origin.FS); ok {
		return c.CdnInfo.PatcherUrl
	}

	scheme := "http"
	if c.CdnInfo.Secure {
		scheme = "https"
	}

	return scheme + "://" + c.CdnInfo.PatcherUrl
}

func (c UniverseConfig) PatcherDir() string {
	if c.VersionDirType == int_netdevil.VersionDirTypeWithVersion {
		return path.Join(c.CdnInfo.PatcherDir, c.Version)
	}
	return c.CdnInfo.PatcherDir
}

type GameInfo struct {
	CrashLogUrl string `xml:"CrashLogUrl"`
}

type PatcherInfo struct {
	ConfigUrl string `xml:"ConfigUrl"`
}

type UniverseEnvironment struct {
	XMLName     xml.Name         `xml:"Environment"`
	GameInfo    GameInfo         `xml:"GameInfo"`
	PatcherInfo PatcherInfo      `xml:"PatcherInfo"`
	Servers     []UniverseConfig `xml:"Servers>Server"`
}
