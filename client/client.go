package client

import (
	"os/exec"
)

type Client interface {
	Run() (*exec.Cmd, error)
	Path() string
	SetPath(path string) error
}

func NewStandardClient() Client {
	return new(standardClient)
}
