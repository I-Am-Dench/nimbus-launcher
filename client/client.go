package client

import (
	"os/exec"
)

type Client interface {
	Path() string
	SetPath(path string) error
	IsValid() bool
	Start() (*exec.Cmd, error)
}

func NewStandardClient() Client {
	return new(standardClient)
}
