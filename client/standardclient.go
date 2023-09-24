package client

import (
	"fmt"
	"os"
)

type standardClient struct {
	path string
	err  error
}

func (client standardClient) Path() string {
	return client.path
}

// This functionality can be expanded upon later
func (client standardClient) Verify() error {
	stats, err := os.Stat(client.path)
	if err != nil {
		return fmt.Errorf("client verify: %v", err)
	}

	if stats.IsDir() {
		return fmt.Errorf("client verify: '%s' is a directory", client.path)
	}

	return nil
}

func (client *standardClient) SetPath(path string) error {
	client.path = path
	client.err = client.Verify()
	return client.err
}

func (client *standardClient) IsValid() bool {
	return client.err == nil
}
