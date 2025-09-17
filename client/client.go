package client

import (
	"os"
	"os/exec"
	"path/filepath"
)

const (
	DefaultDir = "LEGO Software" + string(filepath.Separator) + "LEGO Universe"
	DefaultExe = "client" + string(filepath.Separator) + "legouniverse.exe"
)

type Config struct {
	Directory string `json:"directory"`
	Name      string `json:"name"`
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

func Start(config Config) (*exec.Cmd, error) {
	path := config.ClientPath()

	cmd := exec.Command(path)
	cmd.Dir = filepath.Dir(path)

	return cmd, cmd.Start()
}
