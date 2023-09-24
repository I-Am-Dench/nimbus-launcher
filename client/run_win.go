//go:build windows
// +build windows

package client

import "os/exec"

func (client standardClient) Run() (*exec.Cmd, error) {
	return nil, nil
}
