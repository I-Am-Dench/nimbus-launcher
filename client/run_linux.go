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
	"strconv"
	"strings"
)

const (
	SteamHome = ".steam/steam/"
)

func getProtonVersion(dir string) int {
	data, err := os.ReadFile(filepath.Join(dir, "version"))
	if err != nil {
		return 0
	}

	rawVersion, _, ok := strings.Cut(string(data), " ")
	if !ok {
		return 0
	}

	version, err := strconv.Atoi(rawVersion)
	if err != nil {
		return 0
	}

	return version
}

func FindRecentProtonVersion(dir string) (string, error) {
	versions, err := filepath.Glob(filepath.Join(dir, "Proton*"))
	if err != nil {
		return "", err
	}

	if len(versions) == 0 {
		return "", fmt.Errorf("no proton versions installed")
	}

	sort.Slice(versions, func(i, j int) bool {
		return getProtonVersion(versions[i]) > getProtonVersion(versions[j])
	})

	return filepath.Join(versions[0], "proton"), nil
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

	proton, err := FindRecentProtonVersion(filepath.Join(steam, "steamapps", "common"))
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

	// compatdata is usually found within ".steam/steam/steamapps/compatdata/{appid}", but since
	// LEGO Universe is no longer in service and importing it into steam as an
	// external app appears to generate a random AppId, to guarantee a directory
	// we'll use "{clientDirectory}/.proton".
	//
	// Possible setting for future versions?
	compatdata := filepath.Join(dir, ".proton")

	if err := os.MkdirAll(compatdata, 0755); err != nil {
		return nil, fmt.Errorf("failed to make compatdata path: %w", err)
	}

	cmd := exec.Command(proton, "run", client.path)
	cmd.Dir = dir
	cmd.Env = os.Environ()

	cmd.Env = append(cmd.Env, []string{
		"WINEDLLOVERRIDES=dinput8.dll=n,b",
		"PROTON_USE_WINED3D=1",
		fmt.Sprintf("STEAM_COMPAT_DATA_PATH=%s", compatdata),
		fmt.Sprintf("STEAM_COMPAT_CLIENT_INSTALL_PATH=%s", steam),
	}...)

	cmd.Stderr = os.Stdout

	return cmd, cmd.Start()
}
