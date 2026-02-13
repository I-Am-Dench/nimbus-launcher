package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/I-Am-Dench/goverbuild/encoding/ldf"
	"github.com/I-Am-Dench/goverbuild/models/boot"
	"github.com/I-Am-Dench/nimbus-launcher/app/nldialogs"
	"github.com/I-Am-Dench/nimbus-launcher/app/nlwidgets"
	"github.com/I-Am-Dench/nimbus-launcher/client"
	"github.com/I-Am-Dench/nimbus-launcher/patcher"
	"github.com/I-Am-Dench/nimbus-launcher/patcher/tracker"
)

const (
	TrackerDir = "defaultclients"

	logThreshold = time.Second / 60
)

type LaunchConfig struct {
	DefaultClient             client.Config `json:"defaultClient"`
	CloseOnPlay               bool          `json:"closeOnPlay"`
	ReviewPatchesBeforeUpdate bool          `json:"reviewPatchesBeforeUpdate"`
}

type ProgressBar struct {
	*nlwidgets.ProgressBar
	lastLog time.Time
}

func (p *ProgressBar) SetValue(f float64) {
	fyne.DoAndWait(func() {
		p.ProgressBar.SetValue(f)
	})
}

func (p *ProgressBar) HideProgress() {
	fyne.DoAndWait(p.ProgressBar.HideProgress)
}

func (p *ProgressBar) Infinite() {
	fyne.DoAndWait(p.ProgressBar.Infinite)
}

func (p *ProgressBar) Progress() {
	fyne.DoAndWait(p.ProgressBar.Progress)
}

func (p *ProgressBar) SetText(s string) {
	// NOTE: Having the fyne.DoAndWait here causes a small, but noticeable,
	// slowdown during patching since (from what I've seen) fyne handles
	// its event queue every 15ms. Just calling SetText will cause fyne to
	// yell at you since it's not being called from the main goroutine, and
	// using fyne.Do doesn't change much since fyne still has to catch up
	// with all of the events when it makes the calls for hiding and showing the
	// progress bars at the end of patcher setup.
	//
	// Thought about maybe showing every other or every 5 logs, but for
	// slow patches (i.e. checking a non-cached unpacked client) we actually
	// DO want to show every message, so we would need some kind of toggle
	// to indicate how we want to handle showing logged messages.
	//
	// May revisit this one day, but for the time being, I think this it's
	// a better user experience to have everything logged in sync.
	fyne.DoAndWait(func() { p.ProgressBar.SetText(s) })
}

func (p *ProgressBar) canLog() bool {
	now := time.Now()
	if now.Sub(p.lastLog) < logThreshold {
		return false
	}
	p.lastLog = now
	return true
}

func (p *ProgressBar) Print(a ...any) {
	text := fmt.Sprint(a...)
	if p.canLog() {
		p.SetText(text)
	}
	slog.Info(text)
}

func (p *ProgressBar) Printf(format string, a ...any) {
	text := fmt.Sprintf(format, a...)
	if p.canLog() {
		p.SetText(text)
	}
	slog.Info(text)
}

func (p *ProgressBar) Println(a ...any) {
	text := fmt.Sprint(a...) // Don't added newlines. Makes progress bar look weird
	if p.canLog() {
		p.SetText(text)
	}
	slog.Info(text)
}

type Launcher struct {
	*fyne.Container

	window fyne.Window

	playButton *widget.Button

	profileBinding ProfileBinding
	currentProfile *Profile

	clientPathBinding binding.String
	clientErrorIcon   *widget.Icon

	playingBinding binding.Bool

	ProgressBar

	cancelFunc func()
	playWg     sync.WaitGroup
}

func NewLauncher(window fyne.Window, profileBinding ProfileBinding, playingBinding binding.Bool) *Launcher {
	l := &Launcher{
		window: window,

		profileBinding:    profileBinding,
		clientPathBinding: binding.NewString(),
		clientErrorIcon:   widget.NewIcon(theme.NewErrorThemedResource(theme.ErrorIcon())),

		playingBinding: playingBinding,

		ProgressBar: ProgressBar{ProgressBar: nlwidgets.NewProgressBar()},
	}

	l.playButton = widget.NewButtonWithIcon("Play", theme.MediaPlayIcon(), l.Play)
	l.playButton.Importance = widget.HighImportance

	clientLabel := widget.NewLabelWithData(l.clientPathBinding)
	clientLabel.Alignment = fyne.TextAlignLeading
	clientLabel.TextStyle = fyne.TextStyle{
		Bold: true,
	}
	clientLabel.Truncation = fyne.TextTruncateEllipsis

	l.Container = container.NewBorder(
		l.ProgressBar.Container, nil, l.clientErrorIcon, l.playButton,
		clientLabel,
	)

	AppSettings.Binding.AddListener(l)
	profileBinding.AddListener(l)

	return l
}

func GetAbs(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return strings.ToUpper(abs[:1]) + abs[1:], nil // Capitalizes drive name on Windows
}

func (l *Launcher) ClientConfig() client.Config {
	profile := l.currentProfile
	settings := AppSettings.Get()

	c := settings.Launch.DefaultClient
	if profile != nil && profile.Client != nil {
		if len(profile.Client.Directory) > 0 {
			c.Directory = profile.Client.Directory
		}

		if len(profile.Client.Name) > 0 {
			c.Name = profile.Client.Name
		}

		if profile.Client.IsPacked.HasValue() {
			c.IsPacked = profile.Client.IsPacked.Value
		}

		if len(profile.Client.Locale) > 0 {
			c.Locale = profile.Client.Locale
		}

		if profile.Client.FullDownload.HasValue() {
			c.FullDownload = profile.Client.FullDownload.Value
		}
	}

	installDir, err := GetAbs(c.Directory)
	if err != nil {
		slog.Error(err.Error())
	} else {
		c.Directory = installDir
	}

	return c
}

func (l *Launcher) DataChanged() {
	l.currentProfile, _ = l.profileBinding.Get()

	client := l.ClientConfig()
	l.clientPathBinding.Set(client.ClientPath())

	valid := client.IsValid()
	if valid {
		l.clientErrorIcon.Hide()
	} else {
		l.clientErrorIcon.Show()
	}

	var serverSelected bool
	if l.currentProfile != nil {
		_, serverSelected = l.currentProfile.SelectedServer()
	}

	if playing, _ := l.playingBinding.Get(); !playing && l.currentProfile != nil && (l.currentProfile.Server.Patcher != nil || valid) && serverSelected {
		l.playButton.Enable()
	} else {
		l.playButton.Disable()
	}
}

func (l *Launcher) GetBoot(client client.Config, profile *Profile) (boot.Config, error) {
	server, ok := profile.SelectedServer()
	if !ok {
		return boot.Config{}, errors.New("attempted to launch client without a selected server")
	}

	ctx, cancel := context.WithCancel(context.Background())
	l.cancelFunc = cancel

	l.playWg.Add(1)
	defer func() {
		cancel()
		l.cancelFunc = nil
		l.playWg.Done()
	}()

	tkr, _, err := client.Tracker(TrackerDir)
	if err != nil {
		return boot.Config{}, err
	}
	defer tkr.Close()

	l.SetPatching()
	defer l.HideProgress()

	l.Infinite()

	// We want to run the undoer regardless of whether the profile
	// contains a patcher configuration. This ensures that switching
	// to a profile which doesn't use a patcher will reset the client
	// back to a vanilla state.
	if err := tkr.Undo(ctx); err != nil {
		slog.Error("Failed to run undoer", "error", err)
	}

	if client.IsNewInstall() {
		slog.Info("Found new installation", "dir", client.Directory)
		tkr.SetState(tracker.StateNew)

		if err := os.MkdirAll(client.Directory, 0755); err != nil {
			return boot.Config{}, err
		}
	}

	patcher, err := server.GetPatcher(patcher.Options{
		Log: &l.ProgressBar,

		Locale:           client.Locale,
		FullDownload:     client.FullDownload,
		InstallDirectory: client.Directory,
		ServerId:         profile.Id,
	})
	if err != nil {
		return boot.Config{}, err
	}

	ar, err := patcher.GetVersion(ctx, client.IsPacked)
	if err != nil {
		return boot.Config{}, err
	}
	defer func() {
		if ar != nil {
			if err := ar.Close(); err != nil {
				slog.Error(err.Error())
			}
		}
	}()

	patch, err := patcher.GetPatch(ctx, ar)
	if err != nil {
		return boot.Config{}, err
	}

	l.SetMax(float64(patch.Total()))
	patch.SetProgress(func(n int) {
		l.SetValue(float64(n))
	})

	settings := AppSettings.Get()

	if patch.Total() > 0 && settings.Launch.ReviewPatchesBeforeUpdate && !nldialogs.AskContinuePatch(patch.Summary()) {
		return boot.Config{}, errors.New("patch rejected")
	}

	l.Progress()
	if err := patch.Run(ctx, tkr); err != nil {
		return boot.Config{}, err
	}
	l.Print("Patcher completed!")

	tkr.SetState(tracker.StateNormal)

	storedConfig := profile.Server.BootConfig()

	bootConfig := patcher.GetBoot(client.IsPacked)
	bootConfig.SigninURL = storedConfig.SigninURL
	bootConfig.SignupURL = storedConfig.SignupURL
	bootConfig.PasswordURL = storedConfig.PasswordURL
	bootConfig.RegisterURL = storedConfig.RegisterURL

	return bootConfig, nil
}

func (l *Launcher) ShowError(err error) {
	slog.Error(err.Error())
	fyne.DoAndWait(func() { dialog.ShowError(err, l.window) })
	l.SetNormal()
}

func (l *Launcher) play() {
	l.SetLaunching()

	clientConfig := l.ClientConfig()
	profile := l.currentProfile

	bootConfig, err := l.GetBoot(clientConfig, profile)
	if errors.Is(err, context.Canceled) {
		slog.Info("Launch cancelled")
		l.SetNormal()
		return
	}

	if err != nil {
		slog.Error("Failed to get boot configuration", "error", err)
		fyne.DoAndWait(l.playButton.Disable)
		if !nldialogs.AskContinueOnError(err) {
			l.SetNormal()
			return
		}
		bootConfig = profile.Server.BootConfig()
	}
	fyne.DoAndWait(l.playButton.Disable)

	const mebibyte = 1024 * 1024

	free, used, err := clientConfig.DiskSpace()
	if err != nil {
		slog.Error("Failed to get disk space info", "error", err)
	} else {
		bootConfig.TrackDiskUsage = true
		bootConfig.HDSpaceFree = uint32(free / mebibyte)
		bootConfig.HDSpaceUsed = uint32(used / mebibyte)
		slog.Info("Found disk space info", "freeMB", bootConfig.HDSpaceFree, "usedMB", bootConfig.HDSpaceUsed)
	}

	bootFile, err := os.Create(clientConfig.BootPath())
	if err != nil {
		l.ShowError(err)
		return
	}

	slog.Info("Writing boot config", "serverName", bootConfig.ServerName, "authIP", bootConfig.AuthServerIP, "useCatalog", bootConfig.UseCatalog, "manifestFile", bootConfig.ManifestFile)

	if err := ldf.NewTextEncoder(bootFile).Encode(bootConfig); err != nil {
		l.ShowError(err)
		return
	}

	settings := AppSettings.Get()

	cmd, err := client.Start(clientConfig, !settings.Launch.CloseOnPlay)
	if err != nil {
		l.ShowError(err)
		return
	}

	if settings.Launch.CloseOnPlay {
		fyne.DoAndWait(fyne.CurrentApp().Quit)
		return
	}

	l.SetPlaying()
	go func(cmd *exec.Cmd) {
		if err := cmd.Wait(); err != nil {
			l.ShowError(err)
		}
		slog.Info("Client exited", "exitCode", cmd.ProcessState.ExitCode())
		l.SetNormal()
	}(cmd)
}

func (l *Launcher) Play() {
	go l.play()
}

func (l *Launcher) cancel() {
	if l.cancelFunc != nil {
		l.cancelFunc()
		l.playWg.Wait()
	}
	l.SetNormal()
}

func (l *Launcher) Cancel() {
	l.playButton.Disable()
	go l.cancel()
}

func (l *Launcher) SetNormal() {
	fyne.DoAndWait(func() {
		l.playButton.SetText("Play")
		l.playButton.SetIcon(theme.MediaPlayIcon())
		l.playButton.Importance = widget.HighImportance
		l.playButton.OnTapped = l.Play
		l.playButton.Refresh()
		l.playButton.Enable()

		l.playingBinding.Set(false)
	})
}

func (l *Launcher) SetLaunching() {
	fyne.DoAndWait(func() {
		l.playingBinding.Set(true)
		l.playButton.SetText("Launching...")
		l.playButton.SetIcon(nil)
		l.playButton.Disable()
	})
}

func (l *Launcher) SetPlaying() {
	fyne.DoAndWait(func() {
		l.playButton.SetText("Playing")
		l.playButton.SetIcon(nil)
		l.playButton.Disable()
	})
}

func (l *Launcher) SetPatching() {
	fyne.DoAndWait(func() {
		l.playButton.SetText("Cancel")
		l.playButton.SetIcon(theme.CancelIcon())
		l.playButton.Importance = widget.DangerImportance
		l.playButton.OnTapped = l.Cancel
		l.playButton.Refresh()
		l.playButton.Enable()
	})
}
