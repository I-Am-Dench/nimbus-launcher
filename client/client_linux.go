//go:build linux
// +build linux

package client

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/I-Am-Dench/nimbus-launcher/logger"
)

const (
	DefaultSteamAppId = 21140
)

type Etc = Steam

type ProtonVersion struct {
	Name string
	Path string

	readVersion bool
	version     int
}

func (p *ProtonVersion) Version() (int, error) {
	if p.readVersion {
		return p.version, nil
	}
	p.readVersion = true

	data, err := os.ReadFile(filepath.Join(p.Path, "version"))
	if err != nil {
		return 0, fmt.Errorf("%s: %v", p.Path, err)
	}

	rawVersion, _, ok := bytes.Cut(data, []byte(" "))
	if !ok {
		return 0, fmt.Errorf("%s: no version information", p.Path)
	}

	version, err := strconv.Atoi(string(rawVersion))
	if err != nil {
		return 0, fmt.Errorf("%s: invalid app version: %v", p.Path, err)
	}

	return version, nil
}

type Steam struct {
	Proton     string `json:"proton"`
	SteamHome  string `json:"steamHome"`
	CompatData string `json:"compatdata"`
	UseLog     bool   `json:"useLog"`
	AppId      int64  `json:"appId"`
}

func EvalHomeDir(path string) string {
	if path == "~" {
		home, _ := os.UserHomeDir()
		return home
	}

	if !strings.HasPrefix(path, "~/") {
		return path
	}

	path = strings.TrimLeft(path, "~/")

	home, _ := os.UserHomeDir()
	return filepath.Join(home, path)
}

func (s *Steam) SteamApps() (string, error) {
	path := EvalHomeDir(s.SteamHome)

	steamPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", fmt.Errorf("steamapps: %v", err)
	}

	return filepath.Join(filepath.Join(steamPath, "steamapps", "common")), nil
}

func (s *Steam) FindProton() ([]ProtonVersion, string, error) {
	steamApps, err := s.SteamApps()
	if err != nil {
		return nil, "", fmt.Errorf("proton: %v", err)
	}

	apps, err := filepath.Glob(filepath.Join(steamApps, "Proton*"))
	if err != nil {
		return nil, "", fmt.Errorf("proton: %v", err)
	}

	if len(apps) == 0 {
		return nil, "", errors.New("proton: no proton versions installed")
	}

	versions := []ProtonVersion{}
	for _, path := range apps {
		versions = append(versions, ProtonVersion{
			Name: filepath.Base(path),
			Path: path,
		})
	}

	slices.SortFunc(versions, func(a, b ProtonVersion) int {
		va, err := a.Version()
		if err != nil {
			slog.Error("Failed to find proton version", "error", err)
		}

		vb, err := b.Version()
		if err != nil {
			slog.Error("Failed to find proton version", "error", err)
		}

		return vb - va
	})

	return versions, steamApps, nil
}

func (s *Steam) GetProton(name string) (string, string, error) {
	versions, steamApps, err := s.FindProton()
	if err != nil {
		return "", "", err
	}

	for _, version := range versions {
		if version.Name == name {
			return filepath.Join(version.Path, "proton"), steamApps, nil
		}
	}

	return "", "", fmt.Errorf("could not find proton version: %s", name)
}

func (s *Steam) ProtonOptions() []string {
	versions, _, err := s.FindProton()
	if err != nil {
		slog.Error("Failed to get proton versions", "error", err)
		return []string{}
	}

	options := []string{}
	for _, version := range versions {
		options = append(options, version.Name)
	}
	return options
}

func Start(config Config) (*exec.Cmd, error) {
	proton, steamApps, err := config.Etc.GetProton(config.Etc.Proton)
	if err != nil {
		return nil, err
	}
	slog.Info("Found Proton configuration", "steamapps", steamApps, "proton", proton)

	// compatdata is usually found within ".steam/steam/steamapps/compatdata/{appid}",
	// but since LEGO Universe is no longer in service, and importing it into steam
	// as an external app appears to generate a random AppId, to guarantee a direcotry
	// we'll use Config.Etc.CompatData. (default: {Config.Directory}/.proton)
	compatdata := EvalHomeDir(strings.ReplaceAll(config.Etc.CompatData, "{InstallationDir}", config.Directory))

	if err := os.MkdirAll(compatdata, 0755); err != nil {
		return nil, fmt.Errorf("failed to make compatdata directory: %v", err)
	}

	path := config.ClientPath()

	cmd := exec.Command(proton, "run", path)
	cmd.Dir = filepath.Dir(path)
	cmd.Env = os.Environ()

	cmd.Env = append(cmd.Env,
		"WINEDLLOVERRIDES=\"dinput8.dll=n,b\"",
		"PROTON_USE_WINED3D=1",
		"STEAM_COMPAT_DATA_PATH="+compatdata,
		"STEAM_COMPAT_CLIENT_INSTALL_PATH="+steamApps,
	)

	if config.Etc.AppId > 0 {
		cmd.Env = append(cmd.Env, "SteamGameId="+strconv.FormatInt(config.Etc.AppId, 10))
	}

	if config.Etc.UseLog {
		cmd.Env = append(cmd.Env, "PROTON_LOG=1")
	}

	cmd.Stderr = logger.NewWriter(slog.LevelError)
	cmd.Stdout = logger.NewWriter(slog.LevelInfo)

	slog.Info("Starting client", "cmd", strings.Join(cmd.Args, " "))
	return cmd, cmd.Start()
}
