package netdevil

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"

	"github.com/I-Am-Dench/goverbuild/archive"
	"github.com/I-Am-Dench/goverbuild/archive/manifest"
	"github.com/I-Am-Dench/goverbuild/compress/segmented"
	"github.com/I-Am-Dench/nimbus-launcher/patcher"
	"github.com/I-Am-Dench/nimbus-launcher/patcher/origin"
)

type Downloader struct {
	origin.Resources
	Log patcher.Logger

	Root    string
	TempDir string
}

func (d *Downloader) Open(name string) (*os.File, error) {
	return os.Open(filepath.Join(d.Root, filepath.Clean(name)))
}

func (d *Downloader) Create(name string) (*os.File, error) {
	downloadPath := filepath.Join(d.Root, filepath.Clean(name))

	if err := os.MkdirAll(filepath.Dir(downloadPath), 0755); err != nil {
		return nil, err
	}

	return os.Create(downloadPath)
}

func (d *Downloader) Stat(name string) (os.FileInfo, error) {
	return os.Stat(filepath.Join(d.Root, filepath.Clean(name)))
}

func (d *Downloader) Download(ctx context.Context, source, destination string) (file *os.File, err error) {
	d.Log.Printf("Downloading %s -> %s", source, destination)

	reader, err := d.Get(ctx, source)
	if err != nil {
		return nil, fmt.Errorf("download: %s: %w", source, err)
	}
	defer reader.Close()

	file, err = d.Create(destination)
	if err != nil {
		return nil, fmt.Errorf("download: %s: %w", source, err)
	}

	if _, err := io.Copy(file, reader); err != nil {
		file.Close()
		return nil, fmt.Errorf("download: %s: %w", source, err)
	}

	if cancelled(ctx) {
		file.Close()
		return nil, ctx.Err()
	}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		file.Close()
		return nil, fmt.Errorf("download: %s: %w", source, err)
	}

	return file, nil
}

func (d *Downloader) downloadCompressed(ctx context.Context, entry *manifest.Entry) (r io.ReadCloser, cleanup func() error, err error) {
	hash := hex.EncodeToString(entry.UncompressedChecksum)
	if len(hash) < 2 {
		return nil, nil, fmt.Errorf("download compressed: %s: bad entry checksum: %s", entry.Path, hash)
	}

	source := path.Join(string(hash[0]), string(hash[1]), hash+".sd0")
	tempName := filepath.Join(d.TempDir, filepath.Base(entry.Path)+".sd0")

	temp, err := d.Download(ctx, source, tempName)
	if err != nil {
		return nil, nil, fmt.Errorf("download compressed: %s: %v", entry.Path, err)
	}

	cleanup = func() error {
		return errors.Join(temp.Close(), os.Remove(filepath.Join(d.Root, tempName)))
	}

	checksum := md5.New()
	if _, err := io.Copy(checksum, temp); err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("download compressed: %s: %v", entry.Path, err)
	}

	if sum := checksum.Sum(nil); !bytes.Equal(sum, entry.CompressedChecksum) {
		cleanup()
		return nil, nil, fmt.Errorf("download compressed: %s: mismatched comrpessed checksum: expected %x but got %x", entry.Path, entry.CompressedChecksum, sum)
	}

	if cancelled(ctx) {
		cleanup()
		return nil, nil, ctx.Err()
	}

	if _, err := temp.Seek(0, io.SeekStart); err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("download compressed: %s: %v", entry.Path, err)
	}

	return temp, cleanup, nil
}

func (d *Downloader) DownloadPacked(ctx context.Context, path string, entry *manifest.Entry, archive *archive.Archive) (err error) {
	temp, cleanup, err := d.downloadCompressed(ctx, entry)
	if err != nil {
		return err
	}
	defer func() {
		if e := cleanup(); e != nil && err == nil {
			err = fmt.Errorf("download packed: %s: %v", path, e)
		}
	}()

	pack, record, err := archive.FindPack(path)
	if err != nil {
		return fmt.Errorf("download packed: %s: %v", path, err)
	}

	reader := io.Reader(temp)
	if !record.IsCompressed {
		decompressor, err := segmented.NewDataReader(temp)
		if err != nil {
			return fmt.Errorf("download packed: %s: %v", path, err)
		}
		reader = decompressor
	}

	if err := pack.Store(path, entry.Info, record.IsCompressed, reader); err != nil {
		return fmt.Errorf("download packed: %s: %v", path, err)
	}

	return nil
}

func (d *Downloader) DownloadUnpacked(ctx context.Context, destination string, entry *manifest.Entry) (f *os.File, err error) {
	temp, cleanup, err := d.downloadCompressed(ctx, entry)
	if err != nil {
		return nil, err
	}
	defer func() {
		if e := cleanup(); e != nil && err == nil {
			err = fmt.Errorf("download unpacked: %s: %v", destination, err)
		}
	}()

	file, err := d.Create(destination)
	if err != nil {
		return nil, fmt.Errorf("download unpacked: %s: %w", destination, err)
	}

	decompressor, err := segmented.NewDataReader(temp)
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("download unpacked: %s: %v", destination, err)
	}

	if _, err := io.Copy(file, decompressor); err != nil {
		file.Close()
		return nil, fmt.Errorf("download unpacked: %s: %v", destination, err)
	}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		file.Close()
		return nil, fmt.Errorf("download unpacked: %s: %v", destination, err)
	}

	return file, nil
}
