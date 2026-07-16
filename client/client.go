package client

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"

	"github.com/I-Am-Dench/nimbus-launcher/client/disk"
	"github.com/I-Am-Dench/nimbus-launcher/internal/optional"
	"github.com/I-Am-Dench/nimbus-launcher/patcher"
	"github.com/I-Am-Dench/nimbus-launcher/patcher/tracker"
)

const (
	DefaultDir = "LEGO Software" + string(filepath.Separator) + "LEGO Universe"
	DefaultExe = "client" + string(filepath.Separator) + "legouniverse.exe"
)

type DownloadType = patcher.DownloadType

type Optional struct {
	Directory    string                   `json:"directory,omitempty"`
	Name         string                   `json:"name,omitempty"`
	IsPacked     optional.O[bool]         `json:"packed"`
	Locale       string                   `json:"locale,omitempty"`
	DownloadType optional.O[DownloadType] `json:"downloadType"`
	MaxUgcSpace  optional.O[int]          `json:"maxUgcSpace"`
}

func (o Optional) IsEmpty() bool {
	return len(o.Directory) == 0 && len(o.Name) == 0 && !o.IsPacked.HasValue() && len(o.Locale) == 0 && !o.DownloadType.HasValue() && !o.MaxUgcSpace.HasValue()
}

type Config struct {
	Directory    string       `json:"directory"`
	Name         string       `json:"name"`
	IsPacked     bool         `json:"packed"`
	Locale       string       `json:"locale"`
	DownloadType DownloadType `json:"downloadType"`
	MaxUgcSpace  int          `json:"maxUgcSpace"`
	Etc          Etc          `json:"etc,omitzero"`
}

func (c Config) ClientPath() string {
	return filepath.Join(c.Directory, c.Name)
}

func (c Config) BootPath() string {
	return filepath.Join(filepath.Dir(c.ClientPath()), "boot.cfg")
}

func (c Config) UserMadePath() string {
	return filepath.Join(c.Directory, "client", "res", "BrickModels", "UserMade")
}

func (c Config) ToOptional() Optional {
	maxUgcSize := optional.O[int]{}
	if c.MaxUgcSpace > 0 {
		maxUgcSize = optional.From(c.MaxUgcSpace)
	}

	return Optional{
		Directory:    c.Directory,
		Name:         c.Name,
		IsPacked:     optional.From(c.IsPacked),
		Locale:       c.Locale,
		DownloadType: optional.From(c.DownloadType),
		MaxUgcSpace:  maxUgcSize,
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

func (c Config) Tracker(cacheRoot string) (tracker.Tracker, error) {
	hash, err := tracker.HashFilepath(c.Directory)
	if err != nil {
		return nil, fmt.Errorf("client: %v", err)
	}
	return tracker.New(filepath.Join(cacheRoot, hash), c.Directory)
}

func (c Config) CleanUserMadeModels() error {
	const (
		gibibytes    = 1024 * 1024 * 1024
		manifestName = "manifest.cache"
	)

	userMadePath := c.UserMadePath()

	userMadeManifest, err := ReadUgcManifest(filepath.Join(userMadePath, manifestName))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}

	if err != nil {
		return fmt.Errorf("clean user made models: %v", err)
	}

	keys := slices.Collect(maps.Keys(userMadeManifest))
	slices.SortFunc(keys, func(a, b string) int { return userMadeManifest[a].Compare(userMadeManifest[b]) })

	usedSpace, err := disk.UsedSpace(userMadePath)
	if err != nil {
		return fmt.Errorf("clean user made models: %v", err)
	}

	maxBytes := int64(c.MaxUgcSpace) * gibibytes
	currentSpace := int64(usedSpace)

	for _, path := range keys {
		if currentSpace < maxBytes {
			break
		}

		filePath := filepath.Join(userMadePath, path)

		stat, err := os.Stat(filePath)
		if errors.Is(err, os.ErrNotExist) {
			delete(userMadeManifest, path)
			continue
		}

		if err != nil {
			return fmt.Errorf("clean user made models: %v", err)
		}

		if err := os.Remove(filePath); err != nil {
			return fmt.Errorf("clean user made models: %v", err)
		}

		currentSpace -= stat.Size()
		delete(userMadeManifest, path)
	}

	if err := WriteUgcManifest(filepath.Join(userMadePath, manifestName), userMadeManifest); err != nil {
		return fmt.Errorf("clean user made models: %v", err)
	}
	return nil
}

var DefaultConfig = Config{
	Directory: filepath.Join(GetDefaultAppDirectory(), DefaultDir),
	Name:      DefaultExe,
	IsPacked:  true,
	Locale:    "en_US",
}
