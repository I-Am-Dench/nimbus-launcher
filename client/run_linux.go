//go:build linux
// +build linux

package client

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
)

const (
	SteamHome = ".steam/steam/"

	ProtonAppId = "2805730"
)

var (
	protonPath = ""
	steamPath  = ""
)

func findRecentProtonVersion(dir string) (string, error) {
	versions, err := filepath.Glob(filepath.Join(dir, "Proton*"))
	if err != nil {
		return "", err
	}

	if len(versions) == 0 {
		return "", fmt.Errorf("no proton versions installed")
	}

	sort.Strings(versions)

	return filepath.Join(versions[len(versions)-1], "proton"), nil
}

func init() {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Printf("init: proton: %v", err)
		return
	}

	steam, err := filepath.EvalSymlinks(filepath.Join(home, SteamHome))
	if err != nil {
		log.Printf("init: proton: %v", err)
		return
	}

	log.Printf("init: proton: Found Steam installation at: %s", steam)
	steamPath = steam

	proton, err := findRecentProtonVersion(filepath.Join(steam, "steamapps", "common"))
	if err != nil {
		log.Printf("init: proton: %v", err)
		return
	}

	log.Printf("init: proton: Found Proton installation at: %s", proton)
	protonPath = proton
}

func (client *standardClient) Start() (*exec.Cmd, error) {
	dir := filepath.Dir(client.path)
	compatData := filepath.Join(dir, ".proton")

	os.MkdirAll(compatData, 0755)

	cmd := exec.Command(protonPath, "run", client.path)
	cmd.Dir = dir
	cmd.Env = os.Environ()

	cmd.Env = append(cmd.Env, []string{
		"WINEDLLOVERRIDES=dinput8.dll=n,b",
		"PROTON_USE_WINED3D=1",
		fmt.Sprintf("STEAM_COMPAT_DATA_PATH=%s", compatData),
		fmt.Sprintf("STEAM_COMPAT_CLIENT_INSTALL_PATH=%s", steamPath),
	}...)

	cmd.Stderr = os.Stdout

	return cmd, cmd.Start()
}
