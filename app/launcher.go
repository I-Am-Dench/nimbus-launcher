package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	http_jar "net/http/cookiejar"
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
	"github.com/I-Am-Dench/goverbuild/archive"
	"github.com/I-Am-Dench/goverbuild/encoding/ldf"
	"github.com/I-Am-Dench/goverbuild/models/boot"
	"github.com/I-Am-Dench/nimbus-launcher/app/cookiejar"
	"github.com/I-Am-Dench/nimbus-launcher/app/nlwidgets"
	"github.com/I-Am-Dench/nimbus-launcher/client"
	"github.com/I-Am-Dench/nimbus-launcher/patcher"
	"github.com/I-Am-Dench/nimbus-launcher/patcher/origin"
	"github.com/I-Am-Dench/nimbus-launcher/patcher/undoer"
	"golang.org/x/net/publicsuffix"
)

const (
	UndoerName = "changes.db"

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
	SettingsBinding

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

func NewLauncher(window fyne.Window, settingsBinding SettingsBinding, profileBinding ProfileBinding, playingBinding binding.Bool) *Launcher {
	l := &Launcher{
		SettingsBinding: settingsBinding,

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

	settingsBinding.AddListener(l)
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
	settings := l.Settings()

	c := settings.Launch.DefaultClient
	if profile != nil && profile.Client != nil {
		if len(profile.Client.Directory) > 0 {
			c.Directory = profile.Client.Directory
		}

		if len(profile.Client.Name) > 0 {
			c.Name = profile.Client.Name
		}

		c.IsPacked = profile.Client.IsPacked
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

	if playing, _ := l.playingBinding.Get(); !playing && l.currentProfile != nil && valid {
		l.playButton.Enable()
	} else {
		l.playButton.Disable()
	}
}

func (l *Launcher) getPatcher(ctx context.Context, client client.Config, profile *Profile, jar http.CookieJar) (patcher.Patcher, error) {
	resources, serviceUrl, err := origin.NewResources(profile.Server.Patcher.ServiceUrl)
	if err != nil {
		return nil, err
	}

	// jar, _ := cookiejar.New(&cookiejar.Options{
	// 	PublicSuffixList: publicsuffix.List,
	// })

	if h, ok := resources.(*origin.Http); ok {
		h.Client = &http.Client{
			Jar:       jar,
			Transport: http.DefaultTransport,
		}
	}

	masterIndex, err := profile.Server.Patcher.Environment.GetMasterIndex(ctx, serviceUrl, resources)
	if err != nil {
		return nil, err
	}

	if masterIndex.Config.Type != profile.Server.Patcher.Id {
		return nil, fmt.Errorf("expected patcher %s but Master Index returned %s", profile.Server.Patcher.Id, masterIndex.Config.Type)
	}

	if h, ok := resources.(*origin.Http); ok && len(masterIndex.Authentication) > 0 {
		resources = origin.WithAuthentication(h, AskForCredentials, masterIndex.Authentication)
	}

	return profile.Server.Patcher.Environment.NewPatcher(ctx, patcher.Options{
		Resources: resources,
		Log:       &l.ProgressBar,

		ConfigUrl:         masterIndex.Config.URL,
		AuthenticationUrl: masterIndex.Authentication,
		InstallDirectory:  client.Directory,
		ServerId:          profile.Id,
	})
}

func (l *Launcher) UndoPatches(client client.Config, packed bool) (err error) {
	var arch *archive.Archive
	if packed {
		catalogPath := filepath.Join(client.Directory, patcher.VersionsDir, patcher.CatalogName)

		arch, err = archive.Open(client.Directory, catalogPath)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}

		if err != nil {
			return err
		}
	}
	defer func() {
		if arch != nil {
			if err := arch.Close(); err != nil {
				slog.Error(err.Error())
			}
		}
	}()

	undoer, err := undoer.NewSqlite(UndoerName, client.Directory)
	if err != nil {
		return err
	}

	slog.Info("Running undoer...")
	if err := undoer.Undo(arch); err != nil {
		return err
	}

	return nil
}

func (l *Launcher) GetBoot(client client.Config, profile *Profile) (boot.Config, error) {
	// We want to run the undoer regardless of whether the profile
	// contains a patcher configuration. This ensures that switching
	// to a profile which doesn't use a patcher will reset the client
	// back to a vanilla state.
	if err := l.UndoPatches(client, client.IsPacked); err != nil {
		slog.Error("Failed to run undoer", "error", err)
	}

	if profile.Server.Patcher == nil || len(profile.Server.Patcher.Id) == 0 {
		return profile.Server.BootConfig(), nil
	}

	l.SetPatching()
	defer l.HideProgress()

	l.Infinite()

	ctx, cancel := context.WithCancel(context.Background())
	l.cancelFunc = cancel

	l.playWg.Add(1)
	defer func() {
		cancel()
		l.cancelFunc = nil
		l.playWg.Done()
	}()

	var jar http.CookieJar
	jar, err := cookiejar.New("cookies.json", &cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		slog.Error("Failed to open cookie jar", "error", err)
		jar, _ = http_jar.New(&http_jar.Options{PublicSuffixList: publicsuffix.List})
	}
	defer func() {
		if closer, ok := jar.(io.Closer); ok {
			closer.Close()
		}
	}()

	patcher, err := l.getPatcher(ctx, client, profile, jar)
	if err != nil {
		return boot.Config{}, err
	}

	archive, err := patcher.GetVersion(ctx, profile.Client.IsPacked)
	if err != nil {
		return boot.Config{}, err
	}
	defer func() {
		if archive != nil {
			if err := archive.Close(); err != nil {
				slog.Error(err.Error())
			}
		}
	}()

	undoer, err := undoer.NewSqlite(UndoerName, client.Directory)
	if err != nil {
		return boot.Config{}, err
	}

	patch, err := patcher.GetPatch(ctx, archive)
	if err != nil {
		return boot.Config{}, err
	}

	l.SetMax(float64(patch.Total()))
	patch.SetProgress(func(n int) {
		l.SetValue(float64(n))
	})

	l.Progress()
	if err := patch.Run(ctx, undoer); err != nil {
		return boot.Config{}, err
	}
	l.Print("Patcher completed!")

	storedConfig := profile.Server.BootConfig()

	bootConfig := patcher.GetBoot(client.IsPacked)
	bootConfig.SigninURL = storedConfig.SigninURL
	bootConfig.SignupURL = storedConfig.SignupURL
	bootConfig.PasswordURL = storedConfig.PasswordURL
	bootConfig.RegisterURL = storedConfig.RegisterURL

	return *bootConfig, nil
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
		if !AskContinueOnError(err) {
			l.SetNormal()
			return
		}
		bootConfig = profile.Server.BootConfig()
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

	settings := l.Settings()

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
