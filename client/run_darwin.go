//go:build darwin
// +build darwin

package client

import (
	"errors"
	"os/exec"
)

func (client *standardClient) Start() (*exec.Cmd, error) {
	return nil, errors.New("client start: functionality has not yet been implemented for this system")
}
