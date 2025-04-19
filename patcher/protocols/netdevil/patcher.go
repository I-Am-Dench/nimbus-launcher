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

	"github.com/I-Am-Dench/goverbuild/archive/cache"
	"github.com/I-Am-Dench/goverbuild/archive/catalog"
	"github.com/I-Am-Dench/goverbuild/archive/manifest"
	"github.com/I-Am-Dench/goverbuild/models/boot"
	"github.com/I-Am-Dench/nimbus-launcher/patcher"
	"github.com/I-Am-Dench/nimbus-launcher/patcher/client"
	"github.com/I-Am-Dench/nimbus-launcher/patcher/remote"
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

	ConfigFile = "client/boot.cfg"
)

type Patcher struct {
	UserConfig
	Downloader

	ServerId string

	// patcher.Options

	Server    *Server
	CacheFile *cache.Cache

	// downloader *Downloader
	// ctx context.Context
}

func (patcher *Patcher) GetBoot(packed bool) *boot.Config {
	boot := patcher.Server.Boot(patcher.Locale, packed)
	if patcher.Resources.Scheme() == remote.FileScheme {
		boot.PatchServerIP = "localhost"
	}

	if patcher.FullDownload {
		boot.ManifestFile = ""
	}

	return boot
}

// func (patcher *Patcher) cancelled() bool {
// 	select {
// 	case <-patcher.ctx.Done():
// 		return true
// 	default:
// 		return false
// 	}
// }

func (patcher *Patcher) versions(name string, atRoot ...bool) string {
	if len(atRoot) > 0 && atRoot[0] {
		return filepath.Join(VersionsDir, name)
	} else {
		return filepath.Join(VersionsDir, "nimbus", patcher.ServerId, name)
	}
}

// func (patcher *Patcher) Open(path string, flags int) (*os.File, error) {
// 	installPath := filepath.Join(patcher.InstallDirectory, filepath.FromSlash(path))

// 	if flags&os.O_CREATE != 0 {
// 		if err := os.MkdirAll(filepath.Dir(installPath), 0755); err != nil {
// 			return nil, err
// 		}
// 	}

// 	return os.OpenFile(installPath, flags, 0755)
// }

func (patcher *Patcher) Stat(path string) (os.FileInfo, error) {
	return os.Stat(filepath.Join(patcher.Root, filepath.FromSlash(path)))
}

// func (patcher *Patcher) Download(source, destination string) (file *os.File, err error) {
// 	patcher.Log.Printf("Downloading %s -> %s", source, destination)

// 	resource, err := patcher.Resources.Get(source)
// 	if err != nil {
// 		return nil, fmt.Errorf("download: %s: %w", source, err)
// 	}
// 	defer resource.Close()

// 	file, err = patcher.Open(destination, os.O_CREATE|os.O_TRUNC|os.O_RDWR)
// 	if err != nil {
// 		return nil, fmt.Errorf("download: %s: %w", source, err)
// 	}

// 	if _, err := io.Copy(file, resource); err != nil {
// 		file.Close()
// 		return nil, fmt.Errorf("download: %s: %w", source, err)
// 	}

// 	if patcher.cancelled() {
// 		return nil, patcher.ctx.Err()
// 	}

// 	if _, err := file.Seek(0, io.SeekStart); err != nil {
// 		file.Close()
// 		return nil, fmt.Errorf("download: %s: %w", source, err)
// 	}

// 	return
// }

// func (patcher *Patcher) downloadCompressed(entry *manifest.Entry) (r io.ReadCloser, cleanup func() error, err error) {
// 	hash := hex.EncodeToString(entry.UncompressedChecksum)
// 	if len(hash) < 2 {
// 		return nil, nil, fmt.Errorf("download compressed: %s: bad entry checksum: %s", entry.Path, hash)
// 	}

// 	source := path.Join(string(hash[0]), string(hash[1]), hash+".sd0")
// 	tempname := patcher.versions(filepath.Base(entry.Path) + ".sd0")

// 	temp, err := patcher.Download(source, tempname)
// 	if err != nil {
// 		return nil, nil, err
// 	}

// 	cleanup = func() error {
// 		return errors.Join(temp.Close(), os.Remove(filepath.Join(patcher.InstallDirectory, tempname)))
// 	}

// 	checksum := md5.New()
// 	if _, err := io.Copy(checksum, temp); err != nil {
// 		cleanup()
// 		return nil, nil, fmt.Errorf("download compressed: %w", err)
// 	}

// 	if sum := checksum.Sum(nil); !bytes.Equal(sum, entry.CompressedChecksum) {
// 		cleanup()
// 		return nil, nil, fmt.Errorf("download compressed: %s: mismatched compressed checksum: %x != %x", entry.Path, sum, entry.CompressedChecksum)
// 	}

// 	if patcher.cancelled() {
// 		cleanup()
// 		return nil, nil, patcher.ctx.Err()
// 	}

// 	if _, err := temp.Seek(0, io.SeekStart); err != nil {
// 		cleanup()
// 		return nil, nil, fmt.Errorf("download compressed: %s: %w", entry.Path, err)
// 	}

// 	return temp, cleanup, nil
// }

// func (patcher *Patcher) DownloadPacked(path string, entry *manifest.Entry, archive *client.Archive) error {
// 	temp, cleanup, err := patcher.downloadCompressed(entry)
// 	if err != nil {
// 		return err
// 	}
// 	defer func() {
// 		if e := cleanup(); e != nil && err == nil {
// 			err = fmt.Errorf("download packed: %s: %w", path, e)
// 		}
// 	}()

// 	pack, err := archive.FindPack(path)
// 	if err != nil {
// 		return fmt.Errorf("download packed: %s: %w", path, err)
// 	}

// 	if err := pack.Store(path, entry.Info, true, temp); err != nil {
// 		return fmt.Errorf("download packed: %s: %w", path, err)
// 	}

// 	if patcher.cancelled() {
// 		return patcher.ctx.Err()
// 	}

// 	return nil
// }

// func (patcher *Patcher) DownloadUnpacked(destination string, entry *manifest.Entry) (r io.ReadCloser, err error) {
// 	temp, cleanup, err := patcher.downloadCompressed(entry)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer func() {
// 		if e := cleanup(); e != nil && err == nil {
// 			err = fmt.Errorf("download unpacked: %s: %w", destination, e)
// 		}
// 	}()

// 	file, err := patcher.Open(destination, os.O_CREATE|os.O_TRUNC|os.O_RDWR)
// 	if err != nil {
// 		return nil, fmt.Errorf("download unpacked: %s: %w", entry.Path, err)
// 	}

// 	decompressor, err := segmented.NewDataReader(temp)
// 	if err != nil {
// 		return nil, fmt.Errorf("download unpacked: %s: %w", entry.Path, err)
// 	}

// 	if _, err := io.Copy(file, decompressor); err != nil {
// 		return nil, fmt.Errorf("download unpacked: %s: %w", entry.Path, err)
// 	}

// 	if _, err := file.Seek(0, io.SeekStart); err != nil {
// 		return nil, fmt.Errorf("download unpacked: %s: %w", entry.Path, err)
// 	}

// 	return file, nil
// }

func (patcher *Patcher) DownloadVersions(ctx context.Context, name string, atRoot ...bool) (*os.File, error) {
	file, err := patcher.Download(ctx, path.Join(patcher.Server.Version, name), patcher.versions(name, atRoot...))
	if err != nil {
		return nil, err
	}
	return file, nil
}

func (patcher *Patcher) DownloadManifest(ctx context.Context, name string, atRoot ...bool) (manifestfile *manifest.Manifest, err error) {
	file, err := patcher.DownloadVersions(ctx, name, atRoot...)
	if err != nil {
		return nil, err
	}
	defer func() {
		if e := file.Close(); err == nil {
			err = e
		}
	}()

	manifestfile, err = manifest.Read(file)
	if err != nil {
		return nil, err
	}

	if cancelled(ctx) {
		return nil, ctx.Err()
	}

	if manifestfile.Version != PatcherVersion {
		return nil, fmt.Errorf("download manifest: %s: incompatible manifest version: expected %d but got %d", name, PatcherVersion, manifestfile.Version)
	}

	return
}

func (patcher *Patcher) verifyQuickCheck(path string, file *os.File, entry *manifest.Entry) (bool, error) {
	qc, ok := patcher.CacheFile.Get(path)
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

func (patcher *Patcher) Fetch(ctx context.Context, name, destination string, manifestfile *manifest.Manifest) (io.ReadCloser, error) {
	entry, ok := manifestfile.GetEntry(name)
	if !ok {
		return nil, fmt.Errorf("fetch: %s: missing manifest entry", name)
	}

	file, err := patcher.Open(destination, os.O_RDONLY)
	if errors.Is(err, os.ErrNotExist) {
		return patcher.DownloadUnpacked(ctx, destination, entry)
	}

	if err != nil {
		return nil, fmt.Errorf("fetch: %s: %w", name, err)
	}

	if ok, err := patcher.verifyQuickCheck(destination, file, entry); err != nil {
		file.Close()
		return nil, fmt.Errorf("fetch: %s: %w", name, err)
	} else if ok {
		return file, nil
	} else {
		if err := entry.VerifyUncompressed(file); err != nil {
			file.Close()
			return patcher.DownloadUnpacked(ctx, destination, entry)
		}

		if _, err := file.Seek(0, io.SeekStart); err != nil {
			file.Close()
			return nil, fmt.Errorf("fetch: %s: %w", name, err)
		}

		if cancelled(ctx) {
			file.Close()
			return nil, fmt.Errorf("fetch: %s: %w", name, ctx.Err())
		}
	}

	stat, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("fetch: %s: %w", name, err)
	}

	if err := patcher.CacheFile.Store(destination, stat, entry.Info); err != nil {
		file.Close()
		return nil, fmt.Errorf("fetch: %s: %w", name, err)
	}

	if cancelled(ctx) {
		file.Close()
		return nil, fmt.Errorf("fetch: %s: %w", name, ctx.Err())
	}

	return file, nil
}

func (patcher *Patcher) FetchManifest(ctx context.Context, name string, manifestfile *manifest.Manifest, atRoot ...bool) (m *manifest.Manifest, err error) {
	reader, err := patcher.Fetch(ctx, name, patcher.versions(name, atRoot...), manifestfile)
	if err != nil {
		return nil, err
	}
	defer func() {
		if e := reader.Close(); err == nil {
			err = e
		}
	}()

	m, err = manifest.Read(reader)
	if err != nil {
		return nil, fmt.Errorf("fetch manifest: %s: %w", name, err)
	}

	if cancelled(ctx) {
		return nil, fmt.Errorf("fetch manifest: %s: %w", name, ctx.Err())
	}

	if m.Version != PatcherVersion {
		return nil, fmt.Errorf("fetch manifest: %s: incompatible manifest version: expected %d but got %d", name, PatcherVersion, m.Version)
	}

	return
}

func (patcher *Patcher) needsPackedDownload(ctx context.Context, path string, entry *manifest.Entry, archive *client.Archive) (bool, error) {
	pack, err := archive.FindPack(path)
	if err != nil {
		if errors.Is(err, client.ErrNotCataloged) {
			err = nil
		}
		return false, err
	}

	record, ok := pack.Search(path)
	if !ok {
		return true, nil
	}

	if cancelled(ctx) {
		return false, ctx.Err()
	}

	return !(record.UncompressedSize == entry.UncompressedSize && bytes.Equal(record.UncompressedChecksum, entry.UncompressedChecksum)), nil
}

func (patcher *Patcher) needsUnpackedDownload(ctx context.Context, path string, entry *manifest.Entry) (bool, error) {
	stat, err := patcher.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return true, nil
	}

	if err != nil {
		return false, err
	}

	if qc, ok := patcher.CacheFile.Get(path); ok {
		if err := qc.Check(stat, entry.Info); err == nil {
			return false, nil
		} else {
			patcher.Log.Printf("%s: %v", path, err)
		}
	}

	file, err := patcher.Open(path, os.O_RDONLY)
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

	return false, patcher.CacheFile.Store(path, stat, entry.Info)
}

func (patcher *Patcher) NeedsDownload(ctx context.Context, path string, entry *manifest.Entry, archive ...*client.Archive) (needsDownload bool, err error) {
	defer func() {
		if err != nil {
			return
		}

		if needsDownload {
			patcher.Log.Print(path, " need downloading")
		} else {
			patcher.Log.Print(path, " is ok")
		}
	}()

	if len(archive) > 0 {
		return patcher.needsPackedDownload(ctx, path, entry, archive[0])
	} else {
		return patcher.needsUnpackedDownload(ctx, path, entry)
	}
}

func (patcher *Patcher) shouldIgnore(resource string) bool {
	if strings.EqualFold(resource, ConfigFile) {
		return true
	}

	resource = strings.ToLower(resource)
	if strings.HasSuffix(resource, ".pk") {
		return true
	}

	if strings.Contains(resource, "_loc") && !strings.Contains(resource, path.Join("_loc", strings.ToLower(patcher.Locale))) {
		return true
	}

	if strings.Contains(resource, path.Join("ndaudio", "vo")) && !strings.Contains(resource, path.Join("ndaudio", "vo", strings.ToLower(patcher.Locale))) {
		return true
	}

	return false
}

func (patcher *Patcher) collectEntries(ctx context.Context, name string, index, hotfix *manifest.Manifest, archive ...*client.Archive) ([]*manifest.Entry, error) {
	manifestfile, err := patcher.FetchManifest(ctx, name, index, true)
	if err != nil {
		return nil, err
	}

	entries := []*manifest.Entry{}

	for _, entry := range manifestfile.Entries {
		if cancelled(ctx) {
			return nil, ctx.Err()
		}

		if hotfix != nil {
			if hotfixEntry, ok := hotfix.GetEntry(entry.Path); ok {
				entry = hotfixEntry
			}
		}

		if patcher.shouldIgnore(entry.Path) {
			continue
		}

		needsDownload, err := patcher.NeedsDownload(ctx, entry.Path, entry, archive...)
		if err != nil {
			return nil, err
		}

		if needsDownload {
			patcher.Log.Print(entry.Path, " needs patching")
			entries = append(entries, entry)
		}
	}

	if hotfix != nil {
		for _, hotfixEntry := range hotfix.Entries {
			if cancelled(ctx) {
				return nil, ctx.Err()
			}

			if _, ok := manifestfile.GetEntry(hotfixEntry.Path); !ok {
				needsDownload, err := patcher.NeedsDownload(ctx, hotfixEntry.Path, hotfixEntry, archive...)
				if err != nil {
					return nil, err
				}

				if needsDownload {
					entries = append(entries, hotfixEntry)
				}
			}
		}
	}

	return entries, nil
}

func (patcher *Patcher) fetchCatalog(ctx context.Context, index *manifest.Manifest) (*catalog.Catalog, error) {
	resource, err := patcher.Fetch(ctx, CatalogFile, patcher.versions(CatalogFile, true), index)
	if err != nil {
		return nil, err
	}
	defer resource.Close()

	return catalog.ReadFrom(resource)
}

func (patcher *Patcher) doPacked(ctx context.Context, index, hotfix *manifest.Manifest) (patcher.Patch, error) {
	if cancelled(ctx) {
		return nil, fmt.Errorf("patcher: packed: %w", ctx.Err())
	}

	catalog, err := patcher.fetchCatalog(ctx, index)
	if err != nil {
		return nil, fmt.Errorf("patcher: packed: %w", err)
	}

	if cancelled(ctx) {
		return nil, fmt.Errorf("patcher: packed: %w", ctx.Err())
	}

	archive := client.NewArchive(catalog, patcher.Root)
	defer func() {
		if err := archive.Close(); err != nil {
			patcher.Log.Print(err)
		}
	}()

	entries, err := patcher.collectEntries(ctx, GameFile, index, hotfix, archive)
	if err != nil {
		return nil, fmt.Errorf("patcher: packed: %w", err)
	}

	if cancelled(ctx) {
		return nil, fmt.Errorf("patcher: packed: %w", ctx.Err())
	}

	patcher.Log.Printf("Found %d entries that need patching", len(entries))

	return &Patch{
		Downloader: patcher.Downloader,

		catalog: catalog,
		entries: entries,
	}, nil
}

func (patcher *Patcher) doUnpacked(ctx context.Context, index, hotfix *manifest.Manifest) (patcher.Patch, error) {
	if cancelled(ctx) {
		return nil, fmt.Errorf("patcher: packed: %w", ctx.Err())
	}

	entries, err := patcher.collectEntries(ctx, GameFile, index, hotfix)
	if err != nil {
		return nil, fmt.Errorf("patcher: unpacked: %w", err)
	}

	if cancelled(ctx) {
		return nil, fmt.Errorf("patcher: unpacked: %w", ctx.Err())
	}

	patcher.Log.Printf("Found %d entries that need patching", len(entries))

	return &Patch{
		Downloader: patcher.Downloader,
		entries:    entries,
	}, nil
}

func (patcher *Patcher) initVersions() error {
	stat, err := os.Stat(filepath.Join(patcher.Root, VersionsDir))
	if err == nil {
		if !stat.IsDir() {
			return errors.New("versions must be a directory")
		}
		return nil
	}

	if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	return os.Mkdir(filepath.Join(patcher.Root, VersionsDir), 0755)
}

func (patcher *Patcher) GetPatch(ctx context.Context, packed bool) (patcher.Patch, error) {
	if cancelled(ctx) {
		return nil, fmt.Errorf("patcher: %w", ctx.Err())
	}

	if err := patcher.initVersions(); err != nil {
		return nil, fmt.Errorf("patcher: %w", err)
	}

	var err error
	patcher.CacheFile, err = cache.Open(filepath.Join(patcher.Root, patcher.versions(CacheFile, true)), 32)
	if err != nil {
		return nil, fmt.Errorf("patcher: %w", err)
	}
	defer func() {
		if err := patcher.CacheFile.Close(); err != nil {
			patcher.Log.Print(err)
		}
	}()

	version, err := patcher.DownloadManifest(ctx, VersionFile)
	if err != nil {
		return nil, fmt.Errorf("patcher: %w", err)
	}

	hotfix, err := patcher.DownloadManifest(ctx, HotFixFile, true)
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return nil, fmt.Errorf("patcher: %w", err)
	}

	if err != nil {
		os.Remove(filepath.Join(patcher.Root, patcher.versions(HotFixFile, true)))
	}

	index, err := patcher.FetchManifest(ctx, IndexFile, version)
	if err != nil {
		return nil, fmt.Errorf("patcher: %w", err)
	}

	for _, entry := range version.Entries {
		if cancelled(ctx) {
			return nil, fmt.Errorf("patcher: %w", ctx.Err())
		}

		if entry.Path == IndexFile || strings.Contains(entry.Path, "patcher.ini") || strings.Contains(entry.Path, "lego_universe_install.exe") {
			continue
		}

		reader, err := patcher.Fetch(ctx, entry.Path, patcher.versions(entry.Path), version)
		if err != nil {
			return nil, fmt.Errorf("patcher: %w", err)
		}
		reader.Close()
	}

	if !patcher.FullDownload {
		return &Patch{entries: []*manifest.Entry{}}, nil
	}

	if packed {
		return patcher.doPacked(ctx, index, hotfix)
	} else {
		return patcher.doUnpacked(ctx, index, hotfix)
	}
}
