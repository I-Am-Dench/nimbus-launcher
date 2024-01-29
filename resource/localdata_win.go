//go:build windows
// +build windows

package resource

import (
	"fmt"
	"log"
	"os"
)

func localDataDirectory() (string, error) {
	localAppData, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("cannot find AppData\\Local: %w", err)
	}

	return localAppData, nil
}

func DefaultApplicationsDirectory() string {
	localData, err := localDataDirectory()
	if err != nil {
		log.Println(err)
		log.Println("Using C:\\")
		return "C:\\"
	}

	return localData
}
