//go:build windows
// +build windows

package client

import (
	"os/exec"
	"path/filepath"
)

func (client standardClient) Start() (*exec.Cmd, error) {
	cmd := exec.Command(client.path)
	cmd.Dir = filepath.Dir(client.path)
	return cmd, cmd.Start()
}
