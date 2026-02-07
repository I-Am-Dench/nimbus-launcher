package netdevil

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/I-Am-Dench/goverbuild/archive"
	"github.com/I-Am-Dench/goverbuild/archive/cache"
	"github.com/I-Am-Dench/goverbuild/archive/manifest"
	"github.com/I-Am-Dench/goverbuild/models/boot"
	"github.com/I-Am-Dench/nimbus-launcher/patcher"
)

const (
	PatcherVersion = 82

	VersionsDir = patcher.VersionsDir

	CacheFile   = "quickcheck.txt"
	VersionFile = "version.txt"
	HotFixFile  = "hotfix.txt"
	IndexFile   = "index.txt"
	MinimalFile = "frontend.txt"
	GameFile    = "trunk.txt"

	CatalogFile = patcher.CatalogName
)

type exclude struct {
	Path   string
	Prefix bool
}

type Patcher struct {
	UserConfig
	Downloader

	serverId string

	gameInfo  GameInfo
	server    Server
	cachePath string
	cacheFile *cache.Cache

	packEntries   map[string]manifest.Entry
	packDownloads map[string]manifest.Entry

	index  *manifest.Manifest
	hotfix *manifest.Manifest

	exclude []exclude
}

func (p Patcher) GetBoot(packed bool) boot.Config {
	return boot.Config{
		ServerName:       p.server.Name,
		PatchServerIP:    p.server.CdnInfo.PatcherUrl,
		AuthServerIP:     p.server.AuthenticationIp,
		PatchServerPort:  80,
		Logging:          p.server.LogLevel,
		DataCenterID:     uint32(p.server.DataCenterId),
		PatchServerDir:   p.server.PatcherDir(),
		UGCUse3dServices: p.server.Use3dServices,
		UGCServerIP:      p.server.UgcCdnInfo.PatcherUrl,
		UGCServerDir:     p.server.UgcCdnInfo.PatcherDir,
		CrashLogURL:      p.gameInfo.CrashLogUrl,
		Locale:           p.server.Language,
		ManifestFile:     GameFile,
		UseCatalog:       packed,
	}
}

func (p Patcher) versions(name string, atRoot ...bool) string {
	if len(atRoot) > 0 && atRoot[0] {
		return filepath.Join(VersionsDir, filepath.Clean(name))
	} else {
		return filepath.Join(VersionsDir, "nimbus", p.serverId, filepath.Clean(name))
	}
}

func (p Patcher) initVersions() error {
	dir := filepath.Join(p.Root, VersionsDir)

	stat, err := os.Stat(dir)
	if err == nil {
		if !stat.IsDir() {
			return fmt.Errorf("versions is not a directory: %s", stat.Name())
		}
		return nil
	}

	if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	return os.Mkdir(dir, 0755)
}

func (p Patcher) readCacheFile(name string) (*cache.Cache, error) {
	file, err := os.Open(name)
	if errors.Is(err, os.ErrNotExist) {
		return &cache.Cache{}, nil
	}

	if err != nil {
		return nil, err
	}
	defer file.Close()

	return cache.Read(file, cache.ReadOptions{
		IgnoreUnmarshalErrors: true,
	})
}

func (p Patcher) DownloadVersions(ctx context.Context, name string, atRoot ...bool) (*os.File, error) {
	serverName := name
	if p.server.VersionDirType == VersionDirTypeVersionHotfixOnly {
		serverName = path.Join(p.server.Version, name)
	}

	file, err := p.Download(ctx, serverName, p.versions(name, atRoot...))
	if err != nil {
		return nil, err
	}
	return file, nil
}

func (p Patcher) DownloadManifest(ctx context.Context, name string, atRoot ...bool) (*manifest.Manifest, error) {
	file, err := p.DownloadVersions(ctx, name, atRoot...)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	manifestFile, err := manifest.Read(file)
	if err != nil {
		return nil, err
	}

	if cancelled(ctx) {
		return nil, ctx.Err()
	}

	if manifestFile.Version != PatcherVersion {
		return nil, fmt.Errorf("download manifest: %s: incompatible manifest version: expected %d but got %d", name, PatcherVersion, manifestFile.Version)
	}

	return manifestFile, nil
}

func (p Patcher) verifyQuickCheck(path string, file *os.File, entry manifest.Entry) (bool, error) {
	qc, ok := p.cacheFile.Load(path)
	if !ok {
		return false, nil
	}

	stat, err := file.Stat()
	if err != nil {
		return false, err
	}

	if err := qc.Check(stat, entry.Info); err != nil {
		return false, nil
	}

	return true, nil
}

func (p Patcher) Fetch(ctx context.Context, name, destination string, manifestFile *manifest.Manifest) (*os.File, error) {
	entry, ok := manifestFile.GetEntry(name)
	if !ok {
		return nil, fmt.Errorf("fetch: %s: missing manifest entry", name)
	}

	file, err := p.Open(destination)
	if errors.Is(err, os.ErrNotExist) {
		return p.DownloadUncataloged(ctx, destination, entry)
	}

	if err != nil {
		return nil, fmt.Errorf("fetch: %s: %v", name, err)
	}

	ok, err = p.verifyQuickCheck(destination, file, entry)
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("fetch: %s: %v", name, err)
	}

	if ok {
		return file, nil
	} else {
		if err := entry.VerifyUncompressed(file); err != nil {
			file.Close()
			return p.DownloadUncataloged(ctx, destination, entry)
		}

		if _, err := file.Seek(0, io.SeekStart); err != nil {
			file.Close()
			return nil, fmt.Errorf("fetch: %s: %v", name, err)
		}
	}

	stat, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("fetch: %s: %v", name, err)
	}

	p.cacheFile.Store(destination, stat, entry.Info)

	if cancelled(ctx) {
		file.Close()
		return nil, ctx.Err()
	}

	return file, nil
}

func (p Patcher) FetchManifest(ctx context.Context, name string, manifestFile, hotfixFile *manifest.Manifest, atRoot ...bool) (*manifest.Manifest, map[string]manifest.Entry, error) {
	reader, err := p.Fetch(ctx, name, p.versions(name, atRoot...), manifestFile)
	if err != nil {
		return nil, nil, err
	}
	defer reader.Close()

	fetchedManifest, err := manifest.Read(reader)
	if err != nil {
		return nil, nil, fmt.Errorf("fetch manifest: %s: %v", name, err)
	}

	if cancelled(ctx) {
		return nil, nil, ctx.Err()
	}

	if fetchedManifest.Version != PatcherVersion {
		return nil, nil, fmt.Errorf("fetch manifest: %s: incompatible manifest version: expected %d but got %d", name, PatcherVersion, fetchedManifest.Version)
	}

	added := make(map[string]manifest.Entry)
	if hotfixFile != nil {
		for hotfixEntry := range hotfixFile.All() {
			if _, ok := fetchedManifest.GetEntry(hotfixEntry.Path); !ok {
				added[hotfixEntry.Path] = hotfixEntry
			}
		}

		fetchedManifest.AddEntries(hotfixFile.Entries()...)
	}

	return fetchedManifest, added, nil
}

func (p Patcher) fetchCatalog(ctx context.Context, index *manifest.Manifest) (string, error) {
	catalogPath := p.versions(CatalogFile, true)

	file, err := p.Fetch(ctx, CatalogFile, catalogPath, index)
	if err != nil {
		return "", err
	}
	file.Close()

	return filepath.Join(p.Root, catalogPath), nil
}

func (p Patcher) getPatcherIni(ctx context.Context) (map[string]string, error) {
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

	if err := p.initVersions(); err != nil {
		return nil, fmt.Errorf("patcher: %v", err)
	}
	p.cachePath = filepath.Join(p.Root, p.versions(CacheFile, true))

	var err error
	p.cacheFile, err = p.readCacheFile(p.cachePath)
	if err != nil {
		p.cacheFile = &cache.Cache{}
		p.Log.Printf("Failed to load cache file: %v", err)
	}

	patcherIni, err := p.getPatcherIni(ctx)
	if err != nil {
		p.Log.Print(err)
		patcherIni = map[string]string{}
	}

	osExclude := "win_exclude"
	if runtime.GOOS == "darwin" {
		osExclude = "max_exclude"
	}

	if excludeValue, ok := patcherIni[osExclude]; ok {
		for _, path := range strings.Split(excludeValue, ",") {
			if s := strings.TrimSpace(path); len(s) > 0 {
				if s[len(s)-1] == '*' {
					p.exclude = append(p.exclude, exclude{strings.TrimRight(s, "*"), true})
				} else {
					p.exclude = append(p.exclude, exclude{s, false})
				}
			}
		}
	}

	version, err := p.DownloadManifest(ctx, VersionFile)
	if err != nil {
		return nil, fmt.Errorf("patcher: download version: %w", err)
	}

	p.hotfix, err = p.DownloadManifest(ctx, HotFixFile)
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return nil, fmt.Errorf("patcher: %w", err)
	}

	if err != nil {
		p.Log.Printf("download hotfix: %s", err)
		os.Remove(filepath.Join(p.Root, p.versions(HotFixFile)))
	}

	p.index, _, err = p.FetchManifest(ctx, IndexFile, version, nil)
	if err != nil {
		return nil, fmt.Errorf("patcher: %w", err)
	}

	for entry := range version.All() {
		if cancelled(ctx) {
			return nil, ctx.Err()
		}

		if entry.Path == IndexFile || strings.Contains(entry.Path, "patcher.ini") || strings.Contains(entry.Path, "lego_universe_install.exe") {
			continue
		}

		reader, err := p.Fetch(ctx, entry.Path, p.versions(entry.Path), version)
		if err != nil {
			return nil, fmt.Errorf("patcher: %w", err)
		}
		reader.Close()
	}

	if !packed {
		return nil, nil
	}

	catalogPath, err := p.fetchCatalog(ctx, p.index)
	if err != nil {
		return nil, fmt.Errorf("patcher: %w", err)
	}

	if cancelled(ctx) {
		return nil, ctx.Err()
	}

	archive, err := archive.Open(p.Root, catalogPath)
	if err != nil {
		return nil, fmt.Errorf("patcher: %v", err)
	}

	return archive, nil
}

func (p Patcher) getGameManifests(ctx context.Context, index, hotfix *manifest.Manifest) (downloadManifest, gameManifest *manifest.Manifest, addedHotfix map[string]manifest.Entry, err error) {
	manifestName := MinimalFile
	if p.FullDownload {
		manifestName = GameFile
	}

	downloadManifest, addedHotfix, err = p.FetchManifest(ctx, manifestName, index, hotfix, p.FullDownload) // Only downloads trunk.txt into root
	if err != nil {
		return nil, nil, nil, err
	}

	if p.FullDownload {
		return downloadManifest, downloadManifest, nil, nil
	}

	gameManifest, _, err = p.FetchManifest(ctx, GameFile, index, hotfix, true)
	if err != nil {
		return nil, nil, nil, err
	}

	return downloadManifest, gameManifest, addedHotfix, nil
}

func (p Patcher) needsPackedDownload(ctx context.Context, path string, entry manifest.Entry, addedHotfix map[string]manifest.Entry, arch *archive.Archive) (bool, error) {
	record, err := arch.Load(path)
	if errors.Is(err, archive.ErrNotCataloged) {
		return p.needsUnpackedDownload(ctx, path, entry)
	}

	if errors.Is(err, archive.ErrNotPacked) {
		return true, nil
	}

	if errors.Is(err, os.ErrNotExist) {
		if _, ok := addedHotfix[path]; ok {
			return true, nil
		}

		p.Log.Printf("%s is from a non-existent pack; downloading it", path)

		record, _ := arch.Catalog().Search(path)
		p.packDownloads[record.PackName] = p.packEntries[record.PackName]
		return false, nil // Don't download this entry, it will be downloaded in the pack
	}

	if err != nil {
		return false, fmt.Errorf("check packed download: %w", err)
	}

	if cancelled(ctx) {
		return false, ctx.Err()
	}

	return record.UncompressedSize != entry.UncompressedSize || !bytes.Equal(record.UncompressedChecksum, entry.UncompressedChecksum), nil
}

func (p Patcher) needsUnpackedDownload(ctx context.Context, path string, entry manifest.Entry) (bool, error) {
	stat, err := p.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return true, nil
	}

	if err != nil {
		return false, fmt.Errorf("check unpacked download: %w", err)
	}

	if qc, ok := p.cacheFile.Load(path); ok {
		if err := qc.Check(stat, entry.Info); err == nil {
			return false, nil
		} else {
			p.Log.Printf("%s: %v", path, err)
		}
	}

	file, err := p.Open(path)
	if err != nil {
		return false, fmt.Errorf("check unpacked download: %w", err)
	}
	defer file.Close()

	if err := entry.VerifyUncompressed(file); err != nil {
		return true, nil
	}

	if cancelled(ctx) {
		return false, ctx.Err()
	}

	p.cacheFile.Store(path, stat, entry.Info)
	return false, nil
}

func (p Patcher) NeedsDownload(ctx context.Context, path string, entry manifest.Entry, addedHotfix map[string]manifest.Entry, archive ...*archive.Archive) (needsDownload bool, err error) {
	defer func() {
		if err != nil {
			return
		}

		if needsDownload {
			p.Log.Print(path, " needs download")
		} else {
			p.Log.Print(path, " is ok")
		}
	}()

	if len(archive) > 0 {
		return p.needsPackedDownload(ctx, path, entry, addedHotfix, archive[0])
	} else {
		return p.needsUnpackedDownload(ctx, path, entry)
	}
}

func (p Patcher) shouldIgnore(name string) bool {
	name = strings.ToLower(name)
	if strings.EqualFold(name, "client/boot.cfg") {
		return true
	}

	// Only download packs for During Play (High-Speed)
	if p.FullDownload && strings.HasSuffix(name, ".pk") {
		return true
	}

	if strings.Contains(name, "_loc") && !strings.Contains(name, path.Join("_loc", strings.ToLower(p.Locale))) {
		return true
	}

	ndaudio := p.Locale
	if ndaudio == "en_US" {
		ndaudio = "default"
	}

	if strings.Contains(name, "ndaudio/vo") && !strings.Contains(name, path.Join("ndaudio", "vo", ndaudio)) {
		return true
	}

	for _, exclude := range p.exclude {
		if exclude.Prefix && strings.HasPrefix(name, exclude.Path) {
			return true
		} else if name == exclude.Path {
			return true
		}
	}

	return false
}

func (p Patcher) collectEntries(ctx context.Context, download *manifest.Manifest, addedHotfix map[string]manifest.Entry, archive ...*archive.Archive) ([]manifest.Entry, error) {
	entries := []manifest.Entry{}

	for entry := range download.All() {
		if cancelled(ctx) {
			return nil, ctx.Err()
		}

		if p.shouldIgnore(entry.Path) {
			continue
		}

		// Skip packs already marked for download
		// Prevents packs gettings downloaded twice
		if _, ok := p.packDownloads[entry.Path]; ok {
			continue
		}

		needsDownload, err := p.NeedsDownload(ctx, entry.Path, entry, addedHotfix, archive...)
		if err != nil {
			return nil, err
		}

		if needsDownload {
			entries = append(entries, entry)
		}
	}

	for _, entry := range p.packDownloads {
		entries = append(entries, entry)
	}

	return entries, nil
}

func (p *Patcher) doPacked(ctx context.Context, index, hotfix *manifest.Manifest, archive *archive.Archive) (patcher.Patch, error) {
	if cancelled(ctx) {
		return nil, ctx.Err()
	}

	downloadManifest, gameManifest, addedHotfix, err := p.getGameManifests(ctx, index, hotfix)
	if err != nil {
		return nil, fmt.Errorf("patcher: packed: %w", err)
	}

	for _, packName := range archive.Catalog().PackNames() {
		packEntry, ok := gameManifest.GetEntry(packName)
		if !ok {
			return nil, fmt.Errorf("patcher: packed: game manifest does not contain pack \"%s\"", packName)
		}
		p.packEntries[packName] = packEntry
	}

	entries, err := p.collectEntries(ctx, downloadManifest, addedHotfix, archive)
	if err != nil {
		return nil, fmt.Errorf("patcher: packed: %w", err)
	}

	p.Log.Printf("Found %d entries that need patching", len(entries))

	return &Patch{
		Downloader: p.Downloader,
		archive:    archive,
		entries:    entries,
	}, nil
}

func (p Patcher) doUnpacked(ctx context.Context, index, hotfix *manifest.Manifest) (patcher.Patch, error) {
	if cancelled(ctx) {
		return nil, ctx.Err()
	}

	downloadManifest, _, addedHotfix, err := p.getGameManifests(ctx, index, hotfix)
	if err != nil {
		return nil, fmt.Errorf("patcher: unpacked: %w", err)
	}

	entries, err := p.collectEntries(ctx, downloadManifest, addedHotfix)
	if err != nil {
		return nil, fmt.Errorf("patcher: unpacked: %w", err)
	}

	p.Log.Printf("Found %d entries that need patching", len(entries))

	return &Patch{
		Downloader: p.Downloader,
		entries:    entries,
	}, nil
}

func (p Patcher) GetPatch(ctx context.Context, archive *archive.Archive) (patcher.Patch, error) {
	if cancelled(ctx) {
		return nil, ctx.Err()
	}
	defer func() {
		if p.cacheFile != nil {
			if err := cache.WriteFile(p.cachePath, p.cacheFile); err != nil {
				p.Log.Print(err)
			}
		}
	}()

	if archive != nil {
		return p.doPacked(ctx, p.index, p.hotfix, archive)
	} else {
		return p.doUnpacked(ctx, p.index, p.hotfix)
	}
}
