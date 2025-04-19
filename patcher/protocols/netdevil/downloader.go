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

	"github.com/I-Am-Dench/goverbuild/archive/manifest"
	"github.com/I-Am-Dench/goverbuild/compress/segmented"
	"github.com/I-Am-Dench/nimbus-launcher/patcher"
	"github.com/I-Am-Dench/nimbus-launcher/patcher/client"
	"github.com/I-Am-Dench/nimbus-launcher/patcher/remote"
)

type Downloader struct {
	remote.Resources
	Log patcher.Logger

	Root    string
	TempDir string
}

func (d *Downloader) Open(path string, flags int) (*os.File, error) {
	downloadPath := filepath.Join(d.Root, filepath.FromSlash(path))

	if flags&os.O_CREATE != 0 {
		if err := os.MkdirAll(filepath.Dir(downloadPath), 0755); err != nil {
			return nil, err
		}
	}

	return os.OpenFile(downloadPath, flags, 0755)
}

func (d *Downloader) Download(ctx context.Context, source, destination string) (file *os.File, err error) {
	d.Log.Printf("Downloading %s -> %s", source, destination)

	resource, err := d.Get(ctx, source)
	if err != nil {
		return nil, fmt.Errorf("download: %s: %w", source, err)
	}
	defer resource.Close()

	file, err = d.Open(destination, os.O_CREATE|os.O_TRUNC|os.O_RDWR)
	if err != nil {
		return nil, fmt.Errorf("download: %s: %w", source, err)
	}

	if _, err := io.Copy(file, resource); err != nil {
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

	return
}

func (d *Downloader) downloadCompressed(ctx context.Context, entry *manifest.Entry) (r io.ReadCloser, cleanup func() error, err error) {
	hash := hex.EncodeToString(entry.UncompressedChecksum)
	if len(hash) < 2 {
		return nil, nil, fmt.Errorf("download compressed: %s: bad entry checksum: %s", entry.Path, hash)
	}

	source := path.Join(string(hash[0]), string(hash[1]), hash+".sd0")
	tempname := filepath.Join(d.TempDir, filepath.Base(entry.Path)+".sd0")

	temp, err := d.Download(ctx, source, tempname)
	if err != nil {
		return nil, nil, err
	}

	cleanup = func() error {
		return errors.Join(temp.Close(), os.Remove(filepath.Join(d.Root, tempname)))
	}

	checksum := md5.New()
	if _, err := io.Copy(checksum, temp); err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("download compressed: %w", err)
	}

	if sum := checksum.Sum(nil); !bytes.Equal(sum, entry.CompressedChecksum) {
		cleanup()
		return nil, nil, fmt.Errorf("download compressed: %s: mismatched compressed checksum: %x != %x", entry.Path, sum, entry.CompressedChecksum)
	}

	if cancelled(ctx) {
		cleanup()
		return nil, nil, ctx.Err()
	}

	if _, err := temp.Seek(0, io.SeekStart); err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("download compressed: %s: %w", entry.Path, err)
	}

	return temp, cleanup, nil
}

func (d *Downloader) DownloadPacked(ctx context.Context, path string, entry *manifest.Entry, archive *client.Archive) error {
	temp, cleanup, err := d.downloadCompressed(ctx, entry)
	if err != nil {
		return err
	}
	defer func() {
		if e := cleanup(); e != nil && err == nil {
			err = fmt.Errorf("download packed: %s: %w", path, e)
		}
	}()

	pack, err := archive.FindPack(path)
	if err != nil {
		return fmt.Errorf("download packed: %s: %w", path, err)
	}

	if err := pack.Store(path, entry.Info, true, temp); err != nil {
		return fmt.Errorf("download packed: %s: %w", path, err)
	}

	return nil
}

func (d *Downloader) DownloadUnpacked(ctx context.Context, destination string, entry *manifest.Entry) (r io.ReadCloser, err error) {
	temp, cleanup, err := d.downloadCompressed(ctx, entry)
	if err != nil {
		return nil, err
	}
	defer func() {
		if e := cleanup(); e != nil && err == nil {
			err = fmt.Errorf("download unpacked: %s: %w", destination, e)
		}
	}()

	file, err := d.Open(destination, os.O_CREATE|os.O_TRUNC|os.O_RDWR)
	if err != nil {
		return nil, fmt.Errorf("download unpacked: %s: %w", destination, err)
	}

	decompressor, err := segmented.NewDataReader(temp)
	if err != nil {
		return nil, fmt.Errorf("download unpacked: %s: %w", destination, err)
	}

	if _, err := io.Copy(file, decompressor); err != nil {
		return nil, fmt.Errorf("download unpacked: %s: %w", entry.Path, err)
	}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("download unpacked: %s: %w", entry.Path, err)
	}

	return file, nil
}
