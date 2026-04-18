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
	"github.com/I-Am-Dench/nimbus-launcher/patcher"
	"github.com/I-Am-Dench/nimbus-launcher/patcher/origin"
)

const (
	VersionsDir = patcher.VersionsDir

	CacheFile   = "quickcheck.txt"
	VersionFile = "version.txt"
	HotFixFile  = "hotfix.txt"
	IndexFile   = "index.txt"
	MinimalFile = "frontend.txt"
	GameFile    = "trunk.txt"

	CatalogFile = patcher.CatalogName
)

type VersionDirType int

const (
	VersionDirTypeNone              = VersionDirType(iota) // Only use server dir for requests
	VersionDirTypeVersionHotfixOnly                        // Include version in request path only for getting version.txt and hotfix.txt
	VersionDirTypeWithVersion                              // Include version in request path for all resources
)

type (
	ManifestMap        map[string]manifest.Entry
	VerifyManifestFunc func(*manifest.Manifest) error
)

var errMissingManifestEntry = errors.New("no manifest entry")

type Config struct {
	Locale         string
	DownloadType   patcher.DownloadType
	ServerId       string
	Version        string
	VersionDirType VersionDirType

	VerifyManifestFunc VerifyManifestFunc

	// Allows patch servers to no support frontend.txt.
	// If frontend.txt cannot be found, trunk.txt
	// will be used instead.
	AllowNoMinimalManifest bool
}

type Patcher struct {
	Config
	Downloader

	cachePath string
	cacheFile *cache.Cache

	packEntries   map[string]manifest.Entry
	packDownloads map[string]manifest.Entry

	index  *manifest.Manifest
	hotfix *manifest.Manifest
}

func NewPatcher(config Config, downloader Downloader) *Patcher {
	return &Patcher{
		Config:     config,
		Downloader: downloader,

		packEntries:   make(map[string]manifest.Entry),
		packDownloads: make(map[string]manifest.Entry),
	}
}

func (p Patcher) FullDownload() bool {
	return p.DownloadType == patcher.DownloadTypeFull
}

func (p Patcher) versions(name string, atRoot ...bool) string {
	if len(atRoot) > 0 && atRoot[0] {
		return filepath.Join(VersionsDir, filepath.Clean(name))
	} else {
		return filepath.Join(VersionsDir, "nimbus", p.ServerId, filepath.Clean(name))
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
	if p.VersionDirType == VersionDirTypeVersionHotfixOnly {
		serverName = path.Join(p.Version, name)
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

	if Cancelled(ctx) {
		return nil, ctx.Err()
	}

	if p.VerifyManifestFunc != nil {
		if err := p.VerifyManifestFunc(manifestFile); err != nil {
			return nil, fmt.Errorf("download manifest: %s: %v", name, err)
		}
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
		return nil, fmt.Errorf("fetch: %s: %w", name, errMissingManifestEntry)
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

	if Cancelled(ctx) {
		file.Close()
		return nil, ctx.Err()
	}

	return file, nil
}

func (p Patcher) FetchManifest(ctx context.Context, name string, manifestFile, hotfixFile *manifest.Manifest, atRoot ...bool) (*manifest.Manifest, ManifestMap, error) {
	reader, err := p.Fetch(ctx, name, p.versions(name, atRoot...), manifestFile)
	if err != nil {
		return nil, nil, err
	}
	defer reader.Close()

	fetchedManifest, err := manifest.Read(reader)
	if err != nil {
		return nil, nil, fmt.Errorf("fetch manifest: %s: %v", name, err)
	}

	if Cancelled(ctx) {
		return nil, nil, ctx.Err()
	}

	if p.VerifyManifestFunc != nil {
		if err := p.VerifyManifestFunc(fetchedManifest); err != nil {
			return nil, nil, fmt.Errorf("fetch manifest: %s: %v", name, err)
		}
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

func (p *Patcher) GetVersion(ctx context.Context, packed bool) (*archive.Archive, error) {
	if Cancelled(ctx) {
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

	version, err := p.DownloadManifest(ctx, VersionFile)
	if err != nil {
		return nil, fmt.Errorf("patcher: download version: %w", err)
	}

	p.hotfix, err = p.DownloadManifest(ctx, HotFixFile)
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return nil, fmt.Errorf("patcher: %w", err)
	}

	if errors.Is(err, origin.ErrFileNotFound) {
		p.Log.Printf("download hotfix: %v", err)
		os.Remove(filepath.Join(p.Root, p.versions(HotFixFile)))
	} else if err != nil {
		return nil, fmt.Errorf("patcher: %w", err)
	}

	p.index, _, err = p.FetchManifest(ctx, IndexFile, version, nil)
	if err != nil {
		return nil, fmt.Errorf("patcher: %w", err)
	}

	for entry := range version.All() {
		if Cancelled(ctx) {
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

	if Cancelled(ctx) {
		return nil, ctx.Err()
	}

	ar, err := archive.Open(p.Root, catalogPath)
	if err != nil {
		return nil, fmt.Errorf("patcher: %w", err)
	}
	return ar, nil
}

func (p Patcher) needsUnpackedDownload(ctx context.Context, path string, entry manifest.Entry) (bool, error) {
	stat, err := p.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return true, nil
	}

	if err != nil {
		return false, fmt.Errorf("needs unpacked download: %w", err)
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
		return false, fmt.Errorf("needs unpacked download: %w", err)
	}
	defer file.Close()

	if err := entry.VerifyUncompressed(file); err != nil {
		return true, nil
	}

	if Cancelled(ctx) {
		return false, ctx.Err()
	}

	p.cacheFile.Store(path, stat, entry.Info)
	return false, nil
}

func (p Patcher) needsPackedDownload(ctx context.Context, path string, entry manifest.Entry, addedHotfix ManifestMap, ar *archive.Archive) (bool, error) {
	record, err := ar.Load(path)
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

		record, _ := ar.Catalog().Search(path)
		p.packDownloads[record.PackName] = p.packEntries[record.PackName]
		return false, nil // Don't download this entry, it will be downloaded in the pack
	}

	if err != nil {
		return false, fmt.Errorf("needs packed download: %w", err)
	}

	if Cancelled(ctx) {
		return false, ctx.Err()
	}

	return record.UncompressedSize != entry.UncompressedSize || !bytes.Equal(record.UncompressedChecksum, entry.UncompressedChecksum), nil
}

func (p Patcher) NeedsDownload(ctx context.Context, path string, entry manifest.Entry, addedHotfix ManifestMap, ar ...*archive.Archive) (needsDownload bool, err error) {
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

	if len(ar) > 0 {
		return p.needsPackedDownload(ctx, path, entry, addedHotfix, ar[0])
	} else {
		return p.needsUnpackedDownload(ctx, path, entry)
	}
}

func (p Patcher) shouldIgnore(name string, exclude []string) bool {
	if strings.EqualFold(name, "client/boot.cfg") {
		return true
	}

	// Only download packs for During Play (High-Speed)
	if p.FullDownload() && strings.HasSuffix(name, ".pk") {
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

	for _, exclude := range exclude {
		if matched, _ := path.Match(exclude, name); matched {
			return true
		}
	}

	return false
}

func (p Patcher) collectEntries(ctx context.Context, download *manifest.Manifest, addedHotfix ManifestMap, exclude []string, ar ...*archive.Archive) ([]manifest.Entry, error) {
	entries := []manifest.Entry{}

	for entry := range download.All() {
		if Cancelled(ctx) {
			return nil, ctx.Err()
		}

		if p.shouldIgnore(entry.Path, exclude) {
			continue
		}

		// Skip packs already marked for download.
		// Prevents packs getting download twice.
		if _, ok := p.packDownloads[entry.Path]; ok {
			continue
		}

		needsDownload, err := p.NeedsDownload(ctx, entry.Path, entry, addedHotfix, ar...)
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

func (p Patcher) getGameManifests(ctx context.Context, index, hotfix *manifest.Manifest, fullDownload bool) (downloadManifest, gameManifest *manifest.Manifest, addedHotfix ManifestMap, err error) {
	manifestName := MinimalFile
	if fullDownload {
		manifestName = GameFile
	}

	downloadManifest, addedHotfix, err = p.FetchManifest(ctx, manifestName, index, hotfix, fullDownload) // Only downloads trunk.txt into root
	if err != nil {
		// If we failed to download the minimal manifest file,
		// we'll force the use of trunk.txt.
		if errors.Is(err, errMissingManifestEntry) && !fullDownload && p.AllowNoMinimalManifest {
			p.Log.Print("Minimal manifest file is not supported. Using full game manifest.")
			return p.getGameManifests(ctx, index, hotfix, true)
		}
		return nil, nil, nil, err
	}

	if p.FullDownload() {
		return downloadManifest, downloadManifest, nil, nil
	}

	gameManifest, _, err = p.FetchManifest(ctx, GameFile, index, hotfix, true)
	if err != nil {
		return nil, nil, nil, err
	}

	return downloadManifest, gameManifest, addedHotfix, nil
}

func (p *Patcher) DoUnpacked(ctx context.Context, index, hotfix *manifest.Manifest, exclude []string) (patcher.Patch, error) {
	if Cancelled(ctx) {
		return nil, ctx.Err()
	}

	downloadManifest, _, addedHotfix, err := p.getGameManifests(ctx, index, hotfix, p.FullDownload())
	if err != nil {
		return nil, fmt.Errorf("patcher: unpacked: %w", err)
	}

	entries, err := p.collectEntries(ctx, downloadManifest, addedHotfix, exclude)
	if err != nil {
		return nil, fmt.Errorf("patcher: unpacked: %w", err)
	}

	p.Log.Printf("Found %d entries that need patching", len(entries))

	return &Patch{
		Downloader: p.Downloader,
		entries:    entries,
	}, nil
}

func (p *Patcher) DoPacked(ctx context.Context, index, hotfix *manifest.Manifest, exclude []string, ar *archive.Archive) (patcher.Patch, error) {
	if Cancelled(ctx) {
		return nil, ctx.Err()
	}

	downloadManifest, gameManifest, addedHotfix, err := p.getGameManifests(ctx, index, hotfix, p.FullDownload())
	if err != nil {
		return nil, fmt.Errorf("patcher: packed: %w", err)
	}

	for _, packName := range ar.Catalog().PackNames() {
		packEntry, ok := gameManifest.GetEntry(packName)
		if !ok {
			return nil, fmt.Errorf("patcher: packed: game manifest does not contain pack \"%s\"", packName)
		}
		p.packEntries[packName] = packEntry
	}

	entries, err := p.collectEntries(ctx, downloadManifest, addedHotfix, exclude, ar)
	if err != nil {
		return nil, fmt.Errorf("patcher: packed: %w", err)
	}

	p.Log.Printf("Found %d entries that need patching", len(entries))

	return &Patch{
		Downloader: p.Downloader,
		ar:         ar,
		entries:    entries,
	}, nil
}

func (p *Patcher) GetPatch(ctx context.Context, exclude []string, ar *archive.Archive) (patcher.Patch, error) {
	if Cancelled(ctx) {
		return nil, ctx.Err()
	}
	defer func() {
		if p.cacheFile != nil {
			if err := cache.WriteFile(p.cachePath, p.cacheFile); err != nil {
				p.Log.Print(err)
			}
		}
	}()

	if ar != nil {
		return p.DoPacked(ctx, p.index, p.hotfix, exclude, ar)
	} else {
		return p.DoUnpacked(ctx, p.index, p.hotfix, exclude)
	}
}
