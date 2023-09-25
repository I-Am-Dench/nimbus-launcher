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
	SelectedServer      string `json:"selectedServer"`
	PreviouslyRunServer string `json:"previouslyRunServer"`

	Client struct {
		Directory            string `json:"directory"`
		Name                 string `json:"name"`
		RunCommand           string `json:"runCommand"`
		EnvironmentVariables string `json:"environmentVariables"`
	} `json:"client"`

	CloseOnPlay               bool `json:"closeOnPlay"`
	CheckPatchesAutomatically bool `json:"checkPatchesAutomatically"`
	ReviewPatchBeforeUpdate   bool `json:"reviewPatchBeforeUpdate"`
}

func (settings *Settings) Adjust() {
	if settings.Client.Directory == "%{DEFAULTPATH}%" {
		settings.Client.Directory = filepath.Join(
			DefaultApplicationsDirectory(),
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

	err = os.WriteFile(filepath.Join(settingsDir, "launcher.json"), data, 0755)
	if err != nil {
		return fmt.Errorf("write settings: %v", err)
	}

	return nil
}

func DefaultSettings() Settings {
	s := Settings{}
	s.SelectedServer = ""
	s.Client.Directory = "%{DEFAULTPATH}%"
	s.CloseOnPlay = true
	s.ReviewPatchBeforeUpdate = true

	s.Adjust()
	return s
}
