package nimbus

import (
	"context"

	"github.com/I-Am-Dench/goverbuild/archive"
	"github.com/I-Am-Dench/goverbuild/models/boot"
	"github.com/I-Am-Dench/nimbus-launcher/patcher"
	int_netdevil "github.com/I-Am-Dench/nimbus-launcher/patcher/protocols/internal/netdevil"
)

var _ patcher.Patcher = (*Patcher)(nil)

type Patcher struct {
	patcher *int_netdevil.Patcher
	server  Server
}

func (p Patcher) GetBoot(packed bool) boot.Config {
	ugc := UGC{
		Host:         "localhost",
		Dir:          "3dservices",
		DataCenterId: 150,
	}
	if p.server.UGC.HasValue() {
		ugc = p.server.UGC.Value
	}

	patchPort := int32(80)
	if p.server.Patcher.Port > 0 {
		patchPort = int32(p.server.Patcher.Port)
	}

	manifestFile := ""
	if !p.patcher.FullDownload() {
		manifestFile = int_netdevil.GameFile
	}

	return boot.Config{
		ServerName:       p.server.Name,
		PatchServerIP:    p.server.Patcher.Host,
		AuthServerIP:     p.server.Game.AuthIP,
		PatchServerPort:  patchPort,
		Logging:          100,
		DataCenterID:     uint32(ugc.DataCenterId),
		PatchServerDir:   p.server.Patcher.Dir,
		UGCUse3dServices: p.server.UGC.HasValue(),
		UGCServerIP:      ugc.Host,
		UGCServerDir:     ugc.Dir,
		CrashLogURL:      p.server.Game.CrashLog,
		Locale:           p.patcher.Locale,
		ManifestFile:     manifestFile,
		UseCatalog:       packed,
	}
}

func (p *Patcher) GetVersion(ctx context.Context, packed bool) (*archive.Archive, error) {
	return p.patcher.GetVersion(ctx, packed)
}

func (p *Patcher) GetPatch(ctx context.Context, ar *archive.Archive) (patcher.Patch, error) {
	return p.patcher.GetPatch(ctx, []string{}, ar)
}
