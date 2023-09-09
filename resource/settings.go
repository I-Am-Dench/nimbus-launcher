package resource

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	DEFAULT_EXE_CLIENT = "legouniverse.exe"
)

type Settings struct {
	CurrentServer int `json:"currentServer"`

	Client struct {
		Directory string `json:"directory"`
		Name      string `json:"name"`
	} `json:"client"`
}

func (settings *Settings) Adjust() {
	if settings.Client.Directory == "%{DEFAULTPATH}%" {
		settings.Client.Directory = filepath.Join(
			DefaultAppDataDirectory(),
			DEFAULT_DIR_CLIENT,
		)
	}

	if len(settings.Client.Name) == 0 {
		settings.Client.Name = DEFAULT_EXE_CLIENT
	}
}

func (settings *Settings) ClientPath() string {
	return filepath.Join(settings.Client.Directory, settings.Client.Name)
}

func (settings *Settings) Save() error {
	data, err := json.MarshalIndent(settings, "", "    ")
	if err != nil {
		return fmt.Errorf("marshal settings: %v", err)
	}

	err = os.WriteFile(Of(settingsDir, "launcher.json"), data, 0755)
	if err != nil {
		return fmt.Errorf("write settings: %v", err)
	}

	return nil
}

func DefaultSettings() Settings {
	s := Settings{}
	s.CurrentServer = 0
	s.Client.Directory = "%{DEFAULTPATH}%"

	s.Adjust()
	return s
}
