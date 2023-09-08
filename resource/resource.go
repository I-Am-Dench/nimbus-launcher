package resource

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
)

const (
	assetsDir   = "assets"
	settingsDir = "settings"
	serversDir  = "servers"
)

func Of(dir, name string) string {
	return filepath.Join(dir, name)
}

func Asset(name string) (*fyne.StaticResource, error) {
	bytes, err := os.ReadFile(Of(assetsDir, name))
	if err != nil {
		return nil, fmt.Errorf("assert read error: %v", err)
	}

	return fyne.NewStaticResource(name, bytes), nil
}

func InitializeSettings() error {
	stat, err := os.Stat(settingsDir)
	if !errors.Is(err, os.ErrNotExist) {
		if !stat.IsDir() {
			log.Panicf("\"settings\" already exists as a non-directory: %v\n", err)
		}
		return nil
	}

	log.Println("Initializing settings directories...")

	err = os.MkdirAll(Of(settingsDir, serversDir), 0755)
	if err != nil {
		return err
	}

	settings := DefaultSettings()
	if err := settings.Save(); err != nil {
		return err
	}

	log.Println("Initialization complete.")

	return nil
}

func LauncherSettings() (Settings, error) {
	data, err := os.ReadFile(Of(settingsDir, "launcher.json"))
	if err != nil {
		return Settings{}, fmt.Errorf("launcher settings read: %v", err)
	}

	settings := Settings{}
	err = json.Unmarshal(data, &settings)
	if err != nil {
		return Settings{}, fmt.Errorf("launcher settings unmarshal: %v", err)
	}

	return settings, nil
}
