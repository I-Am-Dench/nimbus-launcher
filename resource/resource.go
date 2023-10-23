package resource

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"

	"fyne.io/fyne/v2"
	"github.com/I-Am-Dench/lu-launcher/client"
	"github.com/I-Am-Dench/lu-launcher/luconfig"
)

const (
	settingsDir = "settings"
	serversDir  = "servers"
)

const (
	DEFAULT_DIR_CLIENT = "LEGO Software/Lego Universe"
)

//go:embed embedded/icon.png
var iconData []byte

var versionPattern = regexp.MustCompile(`^(v|V)?[0-9]+\.[0-9]+\.[0-9]+([0-9a-zA-Z_.-]+)?$`)

func Icon() *fyne.StaticResource {
	return fyne.NewStaticResource("icon.png", iconData)
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
		localServer, err := CreateServer("Localhost", "", "", luconfig.DefaultConfig())
		if err != nil {
			return err
		}

		servers := ServerList{}
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

func PatchRejections() (RejectedPatches, error) {
	rejections := NewRejectedPatches()
	err := rejections.Load()
	return rejections, err
}

func Exists(name string) bool {
	_, err := os.Stat(filepath.Join(settingsDir, name))
	return !errors.Is(err, os.ErrNotExist)
}

func ValidateVersionName(version string) error {
	if !versionPattern.MatchString(version) {
		return fmt.Errorf("invalid version name \"%s\": version name must match `^(v|V)?[0-9]+\\.[0-9]+\\.[0-9]+([0-9a-zA-Z_.-]+)?$`", version)
	}
	return nil
}
