package netdevil

import (
	"context"
	"fmt"
	"runtime"
	"strings"

	"github.com/I-Am-Dench/goverbuild/archive"
	"github.com/I-Am-Dench/goverbuild/models/boot"
	"github.com/I-Am-Dench/nimbus-launcher/patcher"
	int_netdevil "github.com/I-Am-Dench/nimbus-launcher/patcher/protocols/internal/netdevil"
)

var _ patcher.Patcher = (*Patcher)(nil)

type Patcher struct {
	patcher  *int_netdevil.Patcher
	server   Server
	gameInfo GameInfo

	exclude []string
}

func (p Patcher) GetBoot(packed bool) boot.Config {
	manifestFile := ""
	if !p.patcher.FullDownload {
		manifestFile = int_netdevil.GameFile
	}

	return boot.Config{
		ServerName:       p.server.Name,
		PatchServerIP:    p.server.CdnInfo.PatcherUrl,
		AuthServerIP:     p.server.AuthenticationIP,
		PatchServerPort:  80,
		Logging:          p.server.LogLevel,
		DataCenterID:     uint32(p.server.DataCenterId),
		PatchServerDir:   p.server.PatcherDir(),
		UGCUse3dServices: p.server.Use3dServices,
		UGCServerIP:      p.server.UgcCdnInfo.PatcherUrl,
		UGCServerDir:     p.server.UgcCdnInfo.PatcherDir,
		CrashLogURL:      p.gameInfo.CrashLogUrl,
		Locale:           p.server.Language,
		ManifestFile:     manifestFile,
		UseCatalog:       packed,
	}
}

func (p Patcher) getPatcherIni(ctx context.Context) (Ini, error) {
	p.patcher.Log.Printf("Requesting patcher.ini -> %s", p.server.patcherIniUrl)
	r, err := p.server.resources.Get(ctx, p.server.patcherIniUrl)
	if err != nil {
		return nil, fmt.Errorf("get patcher.ini: %w", err)
	}
	defer r.Close()

	return ReadIni(r), nil
}

func (p *Patcher) GetVersion(ctx context.Context, packed bool) (*archive.Archive, error) {
	if cancelled(ctx) {
		return nil, ctx.Err()
	}

	patcherIni, err := p.getPatcherIni(ctx)
	if err != nil {
		p.patcher.Log.Print(err)
		patcherIni = Ini{}
	}

	osExclude := "win_exclude"
	if runtime.GOOS == "darwin" {
		osExclude = "mac_exclude"
	}

	if excludeValue, ok := patcherIni[osExclude]; ok {
		for _, path := range strings.Split(excludeValue, ",") {
			if s := strings.TrimSpace(path); len(s) > 0 {
				p.exclude = append(p.exclude, s)
			}
		}
	}

	return p.patcher.GetVersion(ctx, packed)
}

func (p *Patcher) GetPatch(ctx context.Context, ar *archive.Archive) (patcher.Patch, error) {
	return p.patcher.GetPatch(ctx, p.exclude, ar)
}

func cancelled(ctx context.Context) bool {
	return int_netdevil.Cancelled(ctx)
}
