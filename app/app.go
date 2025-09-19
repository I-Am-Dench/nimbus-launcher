package app

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/I-Am-Dench/goverbuild/models/boot"
	"github.com/I-Am-Dench/nimbus-launcher/version"
)

//go:embed embedded/icon.png
var iconData []byte

type App struct {
	fyne.App

	settingsPath string
	profilesPath string

	settings SettingsBinding
	profiles ProfileListBinding

	profileSelector *ProfileSelector

	main fyne.Window
	info fyne.Window
	sett fyne.Window
}

func New(settingsDir string) (*App, error) {
	a := &App{
		App: app.NewWithID("com.nimbus-launcher"),

		settingsPath: filepath.Join(settingsDir, "settings.json"),
		profilesPath: filepath.Join(settingsDir, "profiles.json"),

		settings: SettingsBinding{binding.NewItem(func(_, _ *Settings) bool { return false })},
		profiles: ProfileListBinding{binding.NewItem(func(_, _ []*Profile) bool { return false })},
	}

	a.settings.AddListener(binding.NewDataListener(func() {
		settings := a.settings.Settings()
		if settings == nil {
			return
		}

		if err := a.WriteSettings(settings); err != nil {
			slog.Error("Failed to write settings", "error", err)
		}
	}))

	settings, err := a.ReadSettings()
	if err != nil {
		return nil, fmt.Errorf("failed to load settings: %v", err)
	}
	a.settings.Set(settings)

	profiles, err := a.ReadProfiles()
	if err != nil {
		return nil, fmt.Errorf("failed to load profiles: %v", err)
	}
	a.profiles.Set(profiles)

	selector, err := NewProfileSelector(a.profiles, a.ShowSettings)
	if err != nil {
		return nil, err
	}
	a.profileSelector = selector

	a.main = a.NewWindow(fmt.Sprint("Nimbus Launcher (", version.Get().Name(), ")"))
	a.main.SetFixedSize(true)
	a.main.Resize(fyne.NewSize(800, 300))
	a.main.SetIcon(fyne.NewStaticResource("icon.png", iconData))
	a.main.SetMaster()

	launcher := NewLauncher(a.main, a.settings, selector.ProfileBinding, selector.PlayingBinding)

	heading := canvas.NewText("Launch LEGO Universe", theme.Color(theme.ColorNameForeground))
	heading.TextSize = 24

	infoButton := widget.NewButtonWithIcon("", theme.InfoIcon(), a.ShowInfo)
	infoButton.Importance = widget.LowImportance

	a.main.SetContent(
		container.NewPadded(
			container.NewBorder(
				container.NewHBox(heading, infoButton),
				launcher.Container,
				nil, nil,
				selector.Container,
			),
		),
	)

	return a, nil
}

func (a *App) ShowInfo() {
	if a.info != nil {
		a.info.Show()
		return
	}

	a.info = NewInfoWindow(a)
	a.info.SetOnClosed(func() { a.info = nil })
	a.info.Show()
}

func (a *App) ShowSettings() {
	if a.sett != nil {
		a.sett.Show()
		return
	}

	a.sett = NewSettingsWindow(a, a.settings, a.profiles, a.profilesPath)
	a.sett.SetOnClosed(func() { a.sett = nil })
	a.sett.CenterOnScreen()
	a.sett.Show()
}

func (a *App) WriteSettings(settings *Settings) error {
	data, err := json.MarshalIndent(settings, "", "    ")
	if err != nil {
		return fmt.Errorf("write settings: %v", err)
	}

	if err := os.WriteFile(a.settingsPath, data, 0755); err != nil {
		return fmt.Errorf("write settings: %v", err)
	}

	return nil
}

func (a *App) ReadSettings() (*Settings, error) {
	data, err := os.ReadFile(a.settingsPath)
	if errors.Is(err, os.ErrNotExist) {
		defaultSettings := DefaultSettings()
		if err := a.WriteSettings(defaultSettings); err != nil {
			return nil, fmt.Errorf("read settings: %v", err)
		}
		return defaultSettings, nil
	}

	if err != nil {
		return nil, fmt.Errorf("read settings: %v", err)
	}

	s := &Settings{}
	if err := json.Unmarshal(data, s); err != nil {
		return nil, fmt.Errorf("read settings: %v", err)
	}

	return s, nil
}

func (a *App) ReadProfiles() ([]*Profile, error) {
	data, err := os.ReadFile(a.profilesPath)
	if errors.Is(err, os.ErrNotExist) {
		profiles := DefaultProfiles(boot.DefaultConfig(), a.profilesPath)

		data, err := json.MarshalIndent(profiles, "", "    ")
		if err != nil {
			return nil, fmt.Errorf("read profiles: %v", err)
		}

		if err := os.WriteFile(a.profilesPath, data, 0664); err != nil {
			return nil, fmt.Errorf("read profiles: %v", err)
		}

		return profiles, nil
	}

	if err != nil {
		return nil, fmt.Errorf("read profiles: %v", err)
	}

	profiles := []*Profile{}
	if err := json.Unmarshal(data, &profiles); err != nil {
		return nil, fmt.Errorf("read profiles: %v", err)
	}

	return profiles, nil
}

func (a *App) Start() {
	a.main.CenterOnScreen()
	a.main.ShowAndRun()
}
