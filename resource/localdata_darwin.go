//go:build darwin
// +build darwin

package resource

import (
	"log"
	"os"
	"path/filepath"
)

func DefaultAppDataDirectory() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		log.Printf("cannot find ~/Library/Application Support: %v\n", err)
		home, _ := os.UserHomeDir()
		desktopDir := filepath.Join(home, "Desktop")
		log.Printf("Using %s\n", desktopDir)
		return desktopDir
	}

	return configDir
}
