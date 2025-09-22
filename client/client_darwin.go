//go:build darwin
// +build darwin

package client

import (
	"errors"
	"os/exec"
)

type Etc = struct{}

func Start(config Config) (*exec.Cmd, error) {
	return nil, errors.ErrUnsupported
}
