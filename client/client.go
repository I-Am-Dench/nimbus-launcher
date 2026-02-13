package client

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/I-Am-Dench/nimbus-launcher/client/disk"
	"github.com/I-Am-Dench/nimbus-launcher/internal/optional"
	"github.com/I-Am-Dench/nimbus-launcher/patcher/tracker"
)

const (
	DefaultDir = "LEGO Software" + string(filepath.Separator) + "LEGO Universe"
	DefaultExe = "client" + string(filepath.Separator) + "legouniverse.exe"
)

type Optional struct {
	Directory    string           `json:"directory"`
	Name         string           `json:"name"`
	IsPacked     optional.O[bool] `json:"packed"`
	Locale       string           `json:"locale,omitempty"`
	FullDownload optional.O[bool] `json:"full_download"`
}

func (o Optional) IsEmpty() bool {
	return len(o.Directory) == 0 && len(o.Name) == 0 && !o.IsPacked.HasValue() && len(o.Locale) == 0 && !o.FullDownload.HasValue()
}

type Config struct {
	Directory    string `json:"directory"`
	Name         string `json:"name"`
	IsPacked     bool   `json:"packed"`
	Locale       string `json:"locale"`
	FullDownload bool   `json:"full_download"`
	Etc          Etc    `json:"etc,omitzero"`
}

func (c Config) ClientPath() string {
	return filepath.Join(c.Directory, c.Name)
}

func (c Config) BootPath() string {
	return filepath.Join(filepath.Dir(c.ClientPath()), "boot.cfg")
}

func (c Config) ToOptional() Optional {
	return Optional{
		Directory:    c.Directory,
		Name:         c.Name,
		IsPacked:     optional.From(c.IsPacked),
		Locale:       c.Locale,
		FullDownload: optional.From(c.FullDownload),
	}
}

func (c Config) DiskSpace() (free, used uint64, err error) {
	free, err = disk.FreeSpace(c.Directory)
	if err != nil {
		return 0, 0, err
	}

	used, err = disk.UsedSpace(filepath.Dir(c.ClientPath()))
	if err != nil {
		return 0, 0, err
	}

	return free, used, nil
}

func (c Config) IsValid() bool {
	stats, err := os.Stat(c.ClientPath())
	if err != nil {
		return false
	}
	return !stats.IsDir()
}

func (c Config) IsNewInstall() bool {
	if _, err := os.Stat(c.Directory); errors.Is(err, os.ErrNotExist) {
		return true
	}

	if _, err := os.Stat(c.ClientPath()); errors.Is(err, os.ErrNotExist) {
		return true
	}

	return false
}

func (c Config) Tracker(cacheRoot string) (tkr tracker.Tracker, hash string, err error) {
	hash, err = tracker.HashFilepath(c.Directory)
	if err != nil {
		return nil, "", fmt.Errorf("client: %v", err)
	}

	tkr, err = tracker.New(filepath.Join(cacheRoot, hash), c.Directory)
	if err != nil {
		return nil, "", fmt.Errorf("client: %v", err)
	}
	return tkr, hash, nil
}

var DefaultConfig = Config{
	Directory: filepath.Join(GetDefaultAppDirectory(), DefaultDir),
	Name:      DefaultExe,
}
