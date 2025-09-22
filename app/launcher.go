package app

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/exec"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/I-Am-Dench/goverbuild/encoding/ldf"
	"github.com/I-Am-Dench/nimbus-launcher/client"
)

type LaunchConfig struct {
	DefaultClient             client.Config `json:"defaultClient"`
	CloseOnPlay               bool          `json:"closeOnPlay"`
	ReviewPatchesBeforeUpdate bool          `json:"reviewPatchesBeforeUpdate"`
}

type Launcher struct {
	*fyne.Container
	SettingsBinding

	window fyne.Window

	playButton *widget.Button

	profileBinding ProfileBinding
	currentProfile *Profile

	clientPathBinding binding.String
	clientErrorIcon   *widget.Icon

	playingBinding binding.Bool

	progress *widget.ProgressBar
	infinite *widget.ProgressBarInfinite

	cancelFunc func()
	playWg     sync.WaitGroup
}

func NewLauncher(window fyne.Window, settingsBinding SettingsBinding, profileBinding ProfileBinding, playingBinding binding.Bool) *Launcher {
	l := &Launcher{
		SettingsBinding: settingsBinding,

		window: window,

		profileBinding:    profileBinding,
		clientPathBinding: binding.NewString(),
		clientErrorIcon:   widget.NewIcon(theme.NewErrorThemedResource(theme.ErrorIcon())),

		playingBinding: playingBinding,

		progress: widget.NewProgressBar(),
		infinite: widget.NewProgressBarInfinite(),
	}

	l.progress.Hide()
	l.infinite.Hide()

	l.playButton = widget.NewButtonWithIcon("Play", theme.MediaPlayIcon(), l.Play)
	l.playButton.Importance = widget.HighImportance

	clientLabel := widget.NewLabelWithData(l.clientPathBinding)
	clientLabel.Alignment = fyne.TextAlignLeading
	clientLabel.TextStyle = fyne.TextStyle{
		Bold: true,
	}
	clientLabel.Truncation = fyne.TextTruncateEllipsis

	l.Container = container.NewBorder(
		container.NewStack(l.progress, l.infinite), nil, l.clientErrorIcon, l.playButton,
		clientLabel,
	)

	settingsBinding.AddListener(l)
	profileBinding.AddListener(l)

	return l
}

func (l *Launcher) HideProgress() {
	l.progress.Hide()
	l.infinite.Hide()
}

func (l *Launcher) Infinite() {
	l.progress.Hide()
	l.infinite.Show()
}

func (l *Launcher) Progress() {
	l.progress.Show()
	l.infinite.Hide()
}

func (l *Launcher) ClientConfig() client.Config {
	profile := l.currentProfile
	settings := l.Settings()

	c := settings.Launch.DefaultClient
	if profile != nil && profile.Client != nil {
		if len(profile.Client.Directory) > 0 {
			c.Directory = profile.Client.Directory
		}

		if len(profile.Client.Name) > 0 {
			c.Name = profile.Client.Name
		}
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

	if playing, _ := l.playingBinding.Get(); !playing && l.currentProfile != nil && valid {
		l.playButton.Enable()
	} else {
		l.playButton.Disable()
	}
}

func (l *Launcher) Patch(ctx context.Context, serverInfo ServerInfo) error {
	l.SetPatching()
	defer l.HideProgress()

	l.Infinite()

	return errors.New("patches are currently disabled")
}

func (l *Launcher) Play() {
	l.SetLaunching()

	ctx, cancel := context.WithCancel(context.Background())
	l.cancelFunc = cancel
	l.playWg.Add(1)
	defer func() {
		cancel()
		l.cancelFunc = nil
		l.playWg.Done()
	}()

	profile := l.currentProfile
	if profile.Server.Patcher != nil && len(profile.Server.Patcher.Id) > 0 {
		if err := l.Patch(ctx, profile.Server); err != nil {
			dialog.ShowError(err, l.window)
		}

		return
	}

	clientConfig := l.ClientConfig()

	bootFile, err := os.Create(clientConfig.BootPath())
	if err != nil {
		dialog.ShowError(err, l.window)
		l.SetNormal()
		return
	}

	settings := l.Settings()

	isPacked := settings.Launch.DefaultClient.IsPacked
	if profile.Client != nil {
		isPacked = profile.Client.IsPacked
	}

	boot := profile.Server.BootConfig()
	boot.UseCatalog = isPacked

	slog.Info("Writing boot config", "serverName", boot.ServerName, "authIP", boot.AuthServerIP, "useCatalog", boot.UseCatalog, "manifestFile", boot.ManifestFile)

	if err := ldf.NewTextEncoder(bootFile).Encode(boot); err != nil {
		dialog.ShowError(err, l.window)
		l.SetNormal()
		return
	}

	cmd, err := client.Start(clientConfig)
	if err != nil {
		dialog.ShowError(err, l.window)
		l.SetNormal()
		return
	}

	if settings.Launch.CloseOnPlay {
		fyne.CurrentApp().Quit()
		return
	}

	l.SetPlaying()
	go func(cmd *exec.Cmd) {
		if err := cmd.Wait(); err != nil {
			dialog.ShowError(err, l.window)
		}
		slog.Info("Client exited", "exitCode", cmd.ProcessState.ExitCode())
		fyne.Do(l.SetNormal)
	}(cmd)
}

func (l *Launcher) Cancel() {
	l.playButton.Disable()
	if l.cancelFunc != nil {
		l.cancelFunc()
		l.playWg.Wait()
	}
	l.SetNormal()
}

func (l *Launcher) SetNormal() {
	l.playButton.SetText("Play")
	l.playButton.SetIcon(theme.MediaPlayIcon())
	l.playButton.Importance = widget.HighImportance
	l.playButton.OnTapped = l.Play
	l.playButton.Refresh()
	l.playButton.Enable()

	l.playingBinding.Set(false)
}

func (l *Launcher) SetLaunching() {
	l.playingBinding.Set(true)

	l.playButton.SetText("Launching...")
	l.playButton.SetIcon(nil)
	l.playButton.Disable()
}

func (l *Launcher) SetPlaying() {
	l.playButton.SetText("Playing")
	l.playButton.SetIcon(nil)
	l.playButton.Disable()
}

func (l *Launcher) SetPatching() {
	l.playButton.SetText("Cancel")
	l.playButton.SetIcon(theme.CancelIcon())
	l.playButton.Importance = widget.DangerImportance
	l.playButton.OnTapped = l.Cancel
	l.playButton.Refresh()
	l.playButton.Enable()
}
