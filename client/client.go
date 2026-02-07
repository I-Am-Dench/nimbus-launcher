package client

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/I-Am-Dench/nimbus-launcher/client/disk"
	"github.com/I-Am-Dench/nimbus-launcher/patcher/tracker"
)

const (
	DefaultDir = "LEGO Software" + string(filepath.Separator) + "LEGO Universe"
	DefaultExe = "client" + string(filepath.Separator) + "legouniverse.exe"
)

type Config struct {
	Directory string `json:"directory"`
	Name      string `json:"name"`
	IsPacked  bool   `json:"packed"`
	Etc       Etc    `json:"etc,omitempty"`
}

func (c *Config) ClientPath() string {
	return filepath.Join(c.Directory, c.Name)
}

func (c *Config) BootPath() string {
	return filepath.Join(filepath.Dir(c.ClientPath()), "boot.cfg")
}

func (c *Config) DiskSpace() (free, used uint64, err error) {
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

func (c *Config) IsValid() bool {
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

func (c Config) Tracker(cacheRoot string) (tracker.Tracker, error) {
	hash, err := tracker.HashFilepath(c.Directory)
	if err != nil {
		return nil, fmt.Errorf("client: %v", err)
	}
	return tracker.New(filepath.Join(cacheRoot, hash), c.Directory)
}

var DefaultConfig = Config{
	Directory: filepath.Join(GetDefaultAppDirectory(), DefaultDir),
	Name:      DefaultExe,
}
