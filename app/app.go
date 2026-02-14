package app

import (
	_ "embed"
	"fmt"
	"log/slog"
	"net/http"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/I-Am-Dench/nimbus-launcher/app/configfile"
	"github.com/I-Am-Dench/nimbus-launcher/version"
)

//go:embed embedded/icon.png
var iconData []byte

type SettingsFile = configfile.ConfigFile[Settings]
type ProfilesFile = configfile.ConfigFile[[]*Profile]

type Preferences interface {
	AppSettings() *SettingsFile
	AppProfiles() *ProfilesFile
}

type App struct {
	fyne.App

	settingsDir string

	settings *SettingsFile
	profiles *ProfilesFile

	profileSelector *ProfileSelectorWidget

	main fyne.Window
	info fyne.Window
	sett fyne.Window
}

func New(settingsDir string, jar http.CookieJar) (*App, error) {
	a := &App{
		App: app.NewWithID("com.nimbus-launcher"),

		settingsDir: settingsDir,

		settings: configfile.New(filepath.Join(settingsDir, "settings.json"), DefaultSettings),
		profiles: configfile.New(filepath.Join(settingsDir, "profiles.json"), DefaultProfiles(DefaultBootConfig(), settingsDir)),
	}

	if _, err := a.settings.Load(); err != nil {
		slog.Error("Failed to load settings", "error", err)
	}

	if _, err := a.profiles.Load(); err != nil {
		slog.Error("Failed to load profiles", "error", err)
	}

	a.main = a.NewWindow(fmt.Sprint("Nimbus Launcher (", version.Get().Name(), ")"))
	a.main.SetFixedSize(true)
	a.main.Resize(fyne.NewSize(800, 300))
	a.main.SetIcon(fyne.NewStaticResource("icon.png", iconData))
	a.main.SetMaster()

	selector, err := NewProfileSelectorWidget(a.main, jar, a, a.ShowSettings)
	if err != nil {
		return nil, err
	}
	a.profileSelector = selector

	launcher := NewLauncherWidget(a.main, a, selector.ProfileBinding, selector.PlayingBinding)

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

func (a *App) AppSettings() *SettingsFile {
	return a.settings
}

func (a *App) AppProfiles() *ProfilesFile {
	return a.profiles
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

	a.sett = NewSettingsWindow(a, a, a.settingsDir)
	a.sett.SetOnClosed(func() { a.sett = nil })
	a.sett.CenterOnScreen()
	a.sett.Show()
}

func (a *App) Start() {
	a.main.CenterOnScreen()
	a.main.ShowAndRun()
}
