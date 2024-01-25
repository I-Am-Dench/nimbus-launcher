package resource

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"github.com/I-Am-Dench/lu-launcher/client"
	"github.com/I-Am-Dench/lu-launcher/ldf"
	"github.com/I-Am-Dench/lu-launcher/resource/patch"
	"github.com/I-Am-Dench/lu-launcher/resource/server"
)

const (
	assetsDir   = "assets"
	settingsDir = "settings"
	serversDir  = "servers"
)

const (
	DEFAULT_DIR_CLIENT = "LEGO Software/Lego Universe"
)

func Asset(name string) (*fyne.StaticResource, error) {
	bytes, err := os.ReadFile(filepath.Join(assetsDir, name))
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
	}

	log.Println("Initializing settings directories...")

	err = os.MkdirAll(filepath.Join(settingsDir, serversDir), 0755)
	if err != nil {
		return err
	}

	if !Exists("launcher.json") {
		log.Println("\"launcher.json\" does not exist; Generating default version")
		settings := DefaultSettings()
		if err := settings.Save(); err != nil {
			return err
		}
		log.Println("Done")
	}

	if !Exists("servers.json") {
		log.Println("\"servers.json\" does not exist; Generating default version")
		localServer, err := CreateServer(server.Config{
			Name:          "Localhost",
			PatchToken:    "",
			PatchProtocol: "",
			Config:        ldf.DefaultBootConfig(),
		})

		if err != nil {
			return err
		}

		servers := ServerList{make([]*server.Server, 1)}
		err = servers.Add(localServer)
		if err != nil {
			return err
		}
		log.Println("Done")
	}

	log.Println("Initialization complete.")

	return nil
}

func LauncherSettings() (Settings, error) {
	data, err := os.ReadFile(filepath.Join(settingsDir, "launcher.json"))
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

func Servers() (ServerList, error) {
	servers := ServerList{}
	err := servers.Load()
	return servers, err
}

func ClientCache() (client.Cache, error) {
	return client.NewSqliteCache(settingsDir)
}

func NewServer(config server.Config) *server.Server {
	config.SettingsDir = settingsDir
	config.DownloadDir = "patches"
	return server.New(config)
}

func CreateServer(config server.Config) (*server.Server, error) {
	server := NewServer(config)
	return server, server.SaveConfig()
}

func PatchRejections() (*patch.RejectionList, error) {
	rejections := patch.NewRejectionList(filepath.Join(settingsDir, "rejectedPatches.json"))
	err := rejections.Load()
	return rejections, err
}

func Exists(name string) bool {
	_, err := os.Stat(filepath.Join(settingsDir, name))
	return !errors.Is(err, os.ErrNotExist)
}
