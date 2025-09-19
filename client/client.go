package client

import (
	"os"
	"path/filepath"
)

const (
	DefaultDir = "LEGO Software" + string(filepath.Separator) + "LEGO Universe"
	DefaultExe = "client" + string(filepath.Separator) + "legouniverse.exe"
)

type Config struct {
	Directory string `json:"directory"`
	Name      string `json:"name"`
	Etc       Etc    `json:"etc,omitempty"`
}

func (c *Config) ClientPath() string {
	return filepath.Join(c.Directory, c.Name)
}

func (c *Config) BootPath() string {
	return filepath.Join(filepath.Dir(c.ClientPath()), "boot.cfg")
}

func (c *Config) IsValid() bool {
	stats, err := os.Stat(c.ClientPath())
	if err != nil {
		return false
	}

	return !stats.IsDir()
}

var DefaultConfig = Config{
	Directory: filepath.Join(GetDefaultAppDirectory(), DefaultDir),
	Name:      DefaultExe,
}
