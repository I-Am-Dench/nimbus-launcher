//go:build windows
// +build windows

package resource

import (
	"fmt"
	"log"
	"os"
)

const (
	DEFAULT_DIR_CLIENT = "LEGO Software\\Lego Universe"
)

func LocalDataDirectory() (string, error) {
	localAppData, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("cannot find AppData\\Local: %v", err)
	}

	return localAppData, nil
}

func DefaultAppDataDirectory() string {
	localData, err := LocalDataDirectory()
	if err != nil {
		log.Println(err)
		log.Println("Using C:\\")
		return "C:\\"
	}

	return localData
}
