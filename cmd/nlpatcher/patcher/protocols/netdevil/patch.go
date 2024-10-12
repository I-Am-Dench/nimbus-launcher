package netdevil

import (
	"bytes"
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/I-Am-Dench/goverbuild/archive/cache"
	"github.com/I-Am-Dench/goverbuild/archive/manifest"
	"github.com/I-Am-Dench/goverbuild/models/boot"
	"github.com/I-Am-Dench/nimbus-launcher/cmd/nlpatcher/patcher"
	"github.com/I-Am-Dench/nimbus-launcher/cmd/nlpatcher/patcher/protocols/netdevil/resources"
)

const (
	PatcherVersion = 10000

	VersionsDir = "versions"

	CacheFile   = "quickcheck.txt"
	VersionFile = "version.txt"
	HotFixFile  = "hotfix.txt"
	IndexFile   = "index.txt"
)

type patchConfig struct {
	patcher.PatchOptions `json:"-"`

	GetResource resources.Func `json:"-"`
	Server      *Server        `json:"-"`

	Locale       string `json:"locale"`
	FullDownload bool   `json:"fullDownload"`
}

type patch struct {
	patchConfig

	CacheFile *cache.Cache
}

func (patch *patch) versions(name string, atRoot ...bool) string {
	if len(atRoot) > 0 && atRoot[0] {
		return filepath.Join(VersionsDir, name)
	} else {
		return filepath.Join(VersionsDir, "nimbus", patch.ServerId, name)
	}
}

func (patch *patch) Open(path string, flags int) (*os.File, error) {
	installPath := filepath.Join(patch.InstallDirectory, path)

	if flags&os.O_CREATE != 0 {
		if err := os.MkdirAll(filepath.Dir(installPath), 0755); err != nil {
			return nil, err
		}
	}

	return os.OpenFile(installPath, flags, 0755)
}

func (patch *patch) Download(source, destination string) (file *os.File, err error) {
	resource, err := patch.GetResource(source)
	if err != nil {
		return nil, fmt.Errorf("download: %s: %w", source, err)
	}
	defer resource.Close()

	file, err = patch.Open(destination, os.O_CREATE|os.O_RDWR|os.O_TRUNC)
	if err != nil {
		return nil, fmt.Errorf("download: %s: %w", source, err)
	}

	if _, err := io.Copy(file, resource); err != nil {
		file.Close()
		return nil, fmt.Errorf("download: %s: %w", source, err)
	}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		file.Close()
		return nil, fmt.Errorf("download: %s: %w", source, err)
	}

	return
}

func (patch *patch) DownloadVersions(name string, atRoot ...bool) (*os.File, error) {
	file, err := patch.Download(path.Join(patch.Server.Version, name), patch.versions(name, atRoot...))
	if err != nil {
		return nil, err
	}

	return file, nil
}

func (patch *patch) DownloadManifest(name string, atRoot ...bool) (manifestfile *manifest.Manifest, err error) {
	file, err := patch.DownloadVersions(name, atRoot...)
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

	if manifestfile.Version != PatcherVersion {
		return nil, fmt.Errorf("download manifest: %s: incompatible manifest versions: expected %d but got %d", name, PatcherVersion, manifestfile.Version)
	}

	return
}

// func (patch *patch) Fetch(source, destination string, manifestfile *manifest.Manifest) (*os.File, error) {
// 	entry, ok := manifestfile.GetEntry(destination)
// 	if !ok {
// 		return nil, fmt.Errorf("fetch: %s: missing manifest entry", source)
// 	}

// 	file, err := patch.Open(destination, os.O_RDONLY)
// 	if errors.Is(err, os.ErrNotExist) {
// 		return patch.Download(source, destination)
// 	}

// 	if err != nil {
// 		return nil, fmt.Errorf("fetch: %s: %w", source, err)
// 	}

// 	if ok, err := patch.verifyQuickCheck(destination, file); err != nil {
// 		return nil, fmt.Errorf("fetch: %s: %w", source, err)
// 	} else if !ok {
// 		if err := patch.verifyUncompressedEntry(file, entry); err != nil {
// 			return patch.Download(source, destination)
// 		}
// 	}

// 	if err := patch.CacheFile.Store(destination, file); err != nil {
// 		return nil, fmt.Errorf("fetch: %s: %w", source, err)
// 	}

// 	if _, err := file.Seek(0, io.SeekStart); err != nil {
// 		return nil, fmt.Errorf("fetch: %s: %w", source, err)
// 	}

// 	return file, nil
// }

func (patch *patch) verifyQuickCheck(path string, file *os.File) (ok bool, err error) {
	defer func() {
		if err == nil {
			_, err = file.Seek(0, io.SeekStart)
		}
	}()

	qc, ok := patch.CacheFile.Get(path)
	if ok {
		if err = qc.Check(file); errors.Is(err, cache.ErrMismatchedQuickCheck) {
			return false, nil
		}
		return err == nil, err
	}

	return false, nil
}

func (patch *patch) verifyUncompressedEntry(file *os.File, entry *manifest.Entry) error {
	stat, err := file.Stat()
	if err != nil {
		return err
	}

	checksum := md5.New()
	if _, err := io.Copy(checksum, file); err != nil {
		return err
	}

	if stat.Size() != entry.UncompressedSize || !bytes.Equal(checksum.Sum(nil), entry.UncompressedChecksum) {
		return errors.New("manifest entry does not match")
	}

	return nil
}

func (patch *patch) FetchVersions(name string, manifest *manifest.Manifest, atRoot ...bool) (*os.File, error) {
	entry, ok := manifest.GetEntry(name)
	if !ok {
		return nil, fmt.Errorf("fetch versions: %s: missing manifest entry", name)
	}

	path := patch.versions(name, atRoot...)

	file, err := patch.Open(path, os.O_RDONLY)
	if errors.Is(err, os.ErrNotExist) {
		return patch.DownloadVersions(name, atRoot...)
	}

	if err != nil {
		return nil, fmt.Errorf("fetch versions: %w", err)
	}

	if ok, err := patch.verifyQuickCheck(path, file); err != nil {
		return nil, fmt.Errorf("fetch versions: %s: %w", name, err)
	} else if !ok {
		if err := patch.verifyUncompressedEntry(file, entry); err != nil {
			return patch.DownloadVersions(name, atRoot...)
		}
	}

	if err := patch.CacheFile.Store(path, file); err != nil {
		return nil, fmt.Errorf("fetch manifest: %s: %w", name, err)
	}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("fetch manifest: %s: %w", name, err)
	}

	return file, nil
}

func (patch *patch) FetchManifest(name string, manifestfile *manifest.Manifest, atRoot ...bool) (m *manifest.Manifest, err error) {
	file, err := patch.FetchVersions(name, manifestfile, atRoot...)
	if err != nil {
		return nil, err
	}
	defer func() {
		if e := file.Close(); err == nil {
			err = e
		}
	}()

	m, err = manifest.Read(file)
	if err != nil {
		return nil, fmt.Errorf("fetch manifest: %s: %w", name, err)
	}

	if m.Version != PatcherVersion {
		return nil, fmt.Errorf("fetch manifest: %s: incompatible manifest versions: expected %d but got %d", name, PatcherVersion, m.Version)
	}

	return
}

func (patch *patch) Patch() (*boot.Config, error) {
	var err error
	patch.CacheFile, err = cache.Open(filepath.Join(patch.InstallDirectory, patch.versions(CacheFile, true)))
	if err != nil {
		return nil, fmt.Errorf("patch: %w", err)
	}

	version, err := patch.DownloadManifest(VersionFile)
	if err != nil {
		return nil, fmt.Errorf("patch: %w", err)
	}

	_, err = patch.DownloadManifest(HotFixFile, true)
	if err != nil {
		os.Remove(filepath.Join(patch.InstallDirectory, patch.versions(HotFixFile, true)))
	}

	_, err = patch.FetchManifest(IndexFile, version)
	if err != nil {
		return nil, fmt.Errorf("patch: %w", err)
	}

	for _, entry := range version.Files {
		if entry.Path == IndexFile || strings.Contains(entry.Path, "patcher.ini") || strings.Contains(entry.Path, "lego_universe_install.exe") {
			continue
		}

		file, err := patch.FetchVersions(entry.Path, version)
		if err != nil {
			return nil, fmt.Errorf("patch: %w", err)
		}
		file.Close()
	}

	return boot.DefaultConfig, nil
}

func newPatch(config patchConfig) (*patch, error) {
	patch := &patch{
		patchConfig: config,
	}

	// cache, err := cache.Open(filepath.Join(config.InstallDirectory, "versions", CacheFile))
	// if err != nil {
	// 	return nil, fmt.Errorf("patch: %w", err)
	// }

	// patch.CacheFile = cache

	return patch, nil
}
