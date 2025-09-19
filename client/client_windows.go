//go:build windows
// +build windows

package client

import (
	"log/slog"
	"os/exec"
	"path/filepath"

	"github.com/I-Am-Dench/nimbus-launcher/logger"
)

type Etc = struct{}

func Start(config Config) (*exec.Cmd, error) {
	path := config.ClientPath()

	cmd := exec.Command(path)
	cmd.Dir = filepath.Dir(path)

	cmd.Stderr = logger.NewWriter(slog.LevelError)

	return cmd, cmd.Start()
}
