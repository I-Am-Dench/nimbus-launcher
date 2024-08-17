//go:build linux
// +build linux

package resource

import (
	"log"
	"os"
	"path/filepath"
)

func DefaultApplicationsDirectory() string {
	homeDirectory, err := os.UserHomeDir()
	if err != nil {
		log.Printf("cannot find home directory: %v\n", err)
		log.Println("Using ~/")
		return "~/"
	}
	return filepath.Join(homeDirectory, "games")
}
