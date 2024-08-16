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

func resolveProton() (string, string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", "", fmt.Errorf("resolve proton: %w", err)
	}

	steam, err := filepath.EvalSymlinks(filepath.Join(home, SteamHome))
	if err != nil {
		return "", "", fmt.Errorf("resolve proton: %w", err)
	}

	proton, err := findRecentProtonVersion(filepath.Join(steam, "steamapps", "common"))
	if err != nil {
		return "", "", fmt.Errorf("resolve proton: %w", err)
	}

	return proton, steam, nil
}

func (client *standardClient) Start() (*exec.Cmd, error) {
	proton, steam, err := resolveProton()
	if err != nil {
		return nil, err
	}
	log.Printf("Found Steam installation: %s", steam)
	log.Printf("Found Proton installation: %s", proton)

	dir := filepath.Dir(client.path)

	// Setting CompatData directory inside of the client's directory
	compatData := filepath.Join(dir, ".proton")

	os.MkdirAll(compatData, 0755)

	cmd := exec.Command(proton, "run", client.path)
	cmd.Dir = dir
	cmd.Env = os.Environ()

	cmd.Env = append(cmd.Env, []string{
		"WINEDLLOVERRIDES=dinput8.dll=n,b",
		"PROTON_USE_WINED3D=1",
		fmt.Sprintf("STEAM_COMPAT_DATA_PATH=%s", compatData),
		fmt.Sprintf("STEAM_COMPAT_CLIENT_INSTALL_PATH=%s", steam),
	}...)

	cmd.Stderr = os.Stdout

	return cmd, cmd.Start()
}
