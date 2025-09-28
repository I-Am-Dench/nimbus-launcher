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
	"strings"

	"github.com/I-Am-Dench/goverbuild/archive"
	"github.com/I-Am-Dench/goverbuild/archive/cache"
	"github.com/I-Am-Dench/goverbuild/archive/manifest"
	"github.com/I-Am-Dench/goverbuild/models/boot"
	"github.com/I-Am-Dench/nimbus-launcher/patcher"
	"github.com/I-Am-Dench/nimbus-launcher/patcher/origin"
)

const (
	PatcherVersion = 10000

	VersionsDir = "versions"

	CacheFile   = "quickcheck.txt"
	VersionFile = "version.txt"
	HotFixFile  = "hotfix.txt"
	IndexFile   = "index.txt"
	GameFile    = "trunk.txt"

	CatalogFile = "primary.pki"
)

type Patcher struct {
	UserConfig
	Downloader

	serverId string

	server    Server
	cachePath string
	cacheFile *cache.Cache

	index  *manifest.Manifest
	hotfix *manifest.Manifest
}

func (p *Patcher) GetBoot(packed bool) *boot.Config {
	bootConfig := p.server.Boot(p.Locale, packed)
	if _, ok := p.Resources.(*origin.FS); ok {
		bootConfig.PatchServerIP = "localhost"
	}

	if p.FullDownload {
		bootConfig.ManifestFile = ""
	}

	return bootConfig
}

func (p *Patcher) versions(name string, atRoot ...bool) string {
	if len(atRoot) > 0 && atRoot[0] {
		return filepath.Join(VersionsDir, filepath.Clean(name))
	} else {
		return filepath.Join(VersionsDir, "nimbus", p.serverId, filepath.Clean(name))
	}
}

func (p *Patcher) initVersions() error {
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

func (p *Patcher) readCacheFile(name string) (*cache.Cache, error) {
	file, err := os.Open(name)
	if errors.Is(err, os.ErrNotExist) {
		return &cache.Cache{}, nil
	}

	if err != nil {
		return nil, err
	}
	defer file.Close()

	return cache.Read(file)
}

func (p *Patcher) DownloadVersions(ctx context.Context, name string, atRoot ...bool) (*os.File, error) {
	file, err := p.Download(ctx, path.Join(p.server.Version, name), p.versions(name, atRoot...))
	if err != nil {
		return nil, err
	}
	return file, nil
}

func (p *Patcher) DownloadManifest(ctx context.Context, name string, atRoot ...bool) (*manifest.Manifest, error) {
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

func (p *Patcher) verifyQuickCheck(path string, file *os.File, entry *manifest.Entry) (bool, error) {
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

func (p *Patcher) Fetch(ctx context.Context, name, destination string, manifestFile *manifest.Manifest) (*os.File, error) {
	entry, ok := manifestFile.GetEntry(name)
	if !ok {
		return nil, fmt.Errorf("fetch: %s: missing manifest entry", name)
	}

	file, err := p.Open(destination)
	if errors.Is(err, os.ErrNotExist) {
		return p.DownloadUnpacked(ctx, destination, entry)
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
			return p.DownloadUnpacked(ctx, destination, entry)
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

func (p *Patcher) FetchManifest(ctx context.Context, name string, manifestFile *manifest.Manifest, atRoot ...bool) (*manifest.Manifest, error) {
	reader, err := p.Fetch(ctx, name, p.versions(name, atRoot...), manifestFile)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	fetchedManifest, err := manifest.Read(reader)
	if err != nil {
		return nil, fmt.Errorf("fetch manifest: %s: %v", name, err)
	}

	if cancelled(ctx) {
		return nil, ctx.Err()
	}

	if fetchedManifest.Version != PatcherVersion {
		return nil, fmt.Errorf("fetch manifest: %s: incompatible manifest version: expected %d but got %d", name, PatcherVersion, fetchedManifest.Version)
	}

	return fetchedManifest, nil
}

func (p *Patcher) needsPackedDownload(ctx context.Context, path string, entry *manifest.Entry, arch *archive.Archive) (bool, error) {
	record, err := arch.Load(path)
	if errors.Is(err, archive.ErrNotCataloged) {
		return false, nil
	}

	if errors.Is(err, archive.ErrNotPacked) {
		return true, nil
	}

	if err != nil {
		return false, err
	}

	if cancelled(ctx) {
		return false, ctx.Err()
	}

	return record.UncompressedSize != entry.UncompressedSize || !bytes.Equal(record.UncompressedChecksum, entry.UncompressedChecksum), nil
}

func (p *Patcher) needsUnpackedDownload(ctx context.Context, path string, entry *manifest.Entry) (bool, error) {
	stat, err := p.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return true, nil
	}

	if err != nil {
		return false, err
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
		return false, err
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

func (p *Patcher) NeedsDownload(ctx context.Context, path string, entry *manifest.Entry, archive ...*archive.Archive) (needsDownload bool, err error) {
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
		return p.needsPackedDownload(ctx, path, entry, archive[0])
	} else {
		return p.needsUnpackedDownload(ctx, path, entry)
	}
}

func (p *Patcher) shouldIgnore(name string) bool {
	if strings.EqualFold(name, "client/boot.cfg") {
		return true
	}

	name = strings.ToLower(name)
	if strings.HasSuffix(name, ".pk") {
		return true
	}

	if strings.Contains(name, "_loc") && !strings.Contains(name, path.Join("_loc", strings.ToLower(p.Locale))) {
		return true
	}

	if strings.Contains(name, path.Join("ndaudio", "vo")) && !strings.Contains(name, path.Join("ndaudio", "vo", strings.ToLower(p.Locale))) {
		return true
	}

	return false
}

func (p *Patcher) collectEntries(ctx context.Context, name string, index, hotfix *manifest.Manifest, archive ...*archive.Archive) ([]*manifest.Entry, error) {
	manifestFile, err := p.FetchManifest(ctx, name, index, true)
	if err != nil {
		return nil, err
	}

	entries := []*manifest.Entry{}

	for _, entry := range manifestFile.Entries {
		if cancelled(ctx) {
			return nil, ctx.Err()
		}

		if hotfix != nil {
			if hotfixEntry, ok := hotfix.GetEntry(entry.Path); ok {
				entry = hotfixEntry
			}
		}

		if p.shouldIgnore(entry.Path) {
			continue
		}

		needsDownload, err := p.NeedsDownload(ctx, entry.Path, entry, archive...)
		if err != nil {
			return nil, err
		}

		if needsDownload {
			p.Log.Print(entry.Path, " needs patching")
			entries = append(entries, entry)
		}
	}

	// Add hotfix entries not present in trunk.txt
	if hotfix != nil {
		for _, hotfixEntry := range hotfix.Entries {
			if cancelled(ctx) {
				return nil, ctx.Err()
			}

			if _, ok := manifestFile.GetEntry(hotfixEntry.Path); ok {
				continue
			}

			needsDownload, err := p.NeedsDownload(ctx, hotfixEntry.Path, hotfixEntry, archive...)
			if err != nil {
				return nil, err
			}

			if needsDownload {
				entries = append(entries, hotfixEntry)
			}
		}
	}

	return entries, nil
}

func (p *Patcher) doPacked(ctx context.Context, index, hotfix *manifest.Manifest, archive *archive.Archive) (patcher.Patch, error) {
	entries, err := p.collectEntries(ctx, GameFile, index, hotfix, archive)
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

func (p *Patcher) doUnpacked(ctx context.Context, index, hotfix *manifest.Manifest) (patcher.Patch, error) {
	if cancelled(ctx) {
		return nil, ctx.Err()
	}

	entries, err := p.collectEntries(ctx, GameFile, index, hotfix)
	if err != nil {
		return nil, fmt.Errorf("patcher: unpacked: %w", err)
	}

	if cancelled(ctx) {
		return nil, ctx.Err()
	}

	p.Log.Printf("Found %d entries that need patching", len(entries))

	return &Patch{
		Downloader: p.Downloader,
		entries:    entries,
	}, nil
}

func (p *Patcher) fetchCatalog(ctx context.Context, index *manifest.Manifest) (string, error) {
	catalogPath := p.versions(CatalogFile, true)

	file, err := p.Fetch(ctx, CatalogFile, catalogPath, index)
	if err != nil {
		return "", err
	}
	file.Close()

	return filepath.Join(p.Root, catalogPath), nil
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
		return nil, fmt.Errorf("patcher: %v", err)
	}

	version, err := p.DownloadManifest(ctx, VersionFile)
	if err != nil {
		return nil, fmt.Errorf("patcher: %w", err)
	}

	p.hotfix, err = p.DownloadManifest(ctx, HotFixFile, true)
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return nil, fmt.Errorf("patcher: %w", err)
	}

	if err != nil {
		os.Remove(filepath.Join(p.Root, p.versions(HotFixFile, true)))
	}

	p.index, err = p.FetchManifest(ctx, IndexFile, version)
	if err != nil {
		return nil, fmt.Errorf("patcher: %w", err)
	}

	for _, entry := range version.Entries {
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

	return &archive, nil
}

func (p *Patcher) GetPatch(ctx context.Context, archive *archive.Archive) (patcher.Patch, error) {
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

	if !p.FullDownload {
		return &Patch{entries: []*manifest.Entry{}}, nil
	}

	if archive != nil {
		return p.doPacked(ctx, p.index, p.hotfix, archive)
	} else {
		return p.doUnpacked(ctx, p.index, p.hotfix)
	}
}
