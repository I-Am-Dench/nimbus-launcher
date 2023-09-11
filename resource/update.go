package resource

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/I-Am-Dench/lu-launcher/luconfig"
)

type Patches struct {
	CurrentVersion string   `json:"currentVersion"`
	Patches        []string `json:"versions"`
}

type Update struct {
	Download []struct {
		Path string `json:"path"`
		Name string `json:"name"`
	} `json:"download"`

	Boot string `json:"boot"`
}

func (update *Update) download(server *Server) error {
	log.Println("Starting downloads...")
	updatePath := filepath.Join("updates", server.Id)
	os.MkdirAll(updatePath, 0755)

	for _, download := range update.Download {
		url, err := url.JoinPath(server.PatchServer, download.Path)
		if err != nil {
			return fmt.Errorf("could not create download url to \"%s\": %v", download.Path, err)
		}

		response, err := http.Get(url)
		if err != nil {
			return fmt.Errorf("could not GET download URL: %v", err)
		}
		defer response.Body.Close()

		if response.StatusCode >= 300 {
			return fmt.Errorf("invalid response status code (%d) from \"%s\"", response.StatusCode, url)
		}

		data, err := io.ReadAll(response.Body)
		if err != nil {
			return fmt.Errorf("could not read body of download response: %v", err)
		}

		err = os.WriteFile(filepath.Join(updatePath, download.Name), data, 0755)
		if err != nil {
			return fmt.Errorf("could not save download \"%s\" to \"%s\": %v", download.Path, download.Name, err)
		}
	}
	return nil
}

func (update *Update) updateBoot(server *Server) error {
	log.Println("Updating boot file...")
	updatePath := filepath.Join("updates", server.Id)

	data, err := os.ReadFile(filepath.Join(updatePath, update.Boot))
	if err != nil {
		return fmt.Errorf("could not read boot patch file \"%s\": %v", update.Boot, err)
	}

	config := luconfig.New()
	err = luconfig.Unmarshal(data, config)
	if err != nil {
		return fmt.Errorf("could not unmarshal boot patch file: %v", err)
	}

	server.Config = config
	return server.SaveConfig()
}

func (update *Update) Run(server *Server) error {
	err := update.download(server)
	if err != nil {
		return err
	}

	if len(update.Boot) > 0 {
		err := update.updateBoot(server)
		if err != nil {
			return err
		}
	}

	return nil
}
