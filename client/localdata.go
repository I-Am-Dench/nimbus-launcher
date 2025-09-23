package client

import (
	"os"
	"path/filepath"
	"runtime"
)

func GetDefaultAppDirectory() string {
	switch runtime.GOOS {
	case "darwin":
		configDir, err := os.UserConfigDir()
		if err != nil {
			home, _ := os.UserHomeDir()
			return filepath.Join(home, "Desktop")
		}
		return configDir
	case "linux":
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "~/"
		}
		return filepath.Join(homeDir, "games")
	case "windows":
		localAppData, err := os.UserCacheDir()
		if err != nil {
			return "C:\\"
		}
		return localAppData
	default:
		return ""
	}
}
