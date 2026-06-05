package app

import (
	"encoding/xml"
	"fmt"
	"log/slog"
	"os"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/I-Am-Dench/nimbus-launcher/app/nlwidgets"
	"github.com/I-Am-Dench/nimbus-launcher/client"
)

type Settings struct {
	Launch LaunchConfig `json:"launch"`
}

func compareProfiles(a, b *Profile) bool {
	return a.Id == b.Id
}

func profileName(p *Profile) string {
	return p.Name
}

type profileSettingsWidget struct {
	*fyne.Container
	Preferences
	window fyne.Window

	settingsDir string

	profileSelector  *nlwidgets.ItemSelector[*Profile]
	listContainer    *fyne.Container
	patcherContainer *fyne.Container
	patcherFunc      PatcherFunc

	content *fyne.Container
}

func newProfileSettingsWidget(window fyne.Window, preferences Preferences, settingsDir string) *profileSettingsWidget {
	p := &profileSettingsWidget{
		Preferences:      preferences,
		window:           window,
		settingsDir:      settingsDir,
		patcherContainer: container.NewStack(),
		content:          container.NewStack(),
	}

	clientSettings := nlwidgets.NewClientSettings(window, false)

	clientSettingsOverride := widget.NewAccordionItem(
		"Client (Override Default)",
		widget.NewForm(clientSettings.Form()...),
	)

	// === Selector === //
	p.profileSelector = nlwidgets.NewItemSelector(p.AppProfiles().Get(), profileName, compareProfiles)
	p.profileSelector.OnChanged = func(profile *Profile) {
		if profile == nil {
			return
		}

		p.SetPatcherSettings(profile)

		if profile.Client == nil {
			clientSettings.Set(client.Optional{})
		} else {
			clientSettings.Set(*profile.Client)
		}

		clientSettingsOverride.Open = profile.Client != nil
	}
	p.profileSelector.PlaceHolder = "(Select server)"
	p.profileSelector.SetSelectedIndex(0)

	addServerButton := widget.NewButtonWithIcon("Add Server", theme.ContentAddIcon(), p.ShowNewProfile)
	addServerButton.Importance = widget.LowImportance

	// === Menu === //
	editItem := fyne.NewMenuItem("Edit", p.ShowEditProfile)
	editItem.Icon = theme.DocumentCreateIcon()

	removeItem := fyne.NewMenuItem("Remove", func() {
		index := p.profileSelector.SelectedIndex()
		profiles := p.AppProfiles().Get()

		profile := profiles[index]
		if profile == nil {
			return
		}

		dialog.NewCustomConfirm(
			"Remove Server",
			"Remove", "Cancel",
			widget.NewLabel(fmt.Sprintf("Remove profile %q?", profile.Name)),
			func(ok bool) {
				if !ok {
					return
				}

				profiles = append(profiles[:index], profiles[index+1:]...)
				if err := os.Remove(profile.Server.Boot); err != nil {
					slog.Error("Failed to remove boot config", "boot", profile.Server.Boot, "error", err)
				}

				if err := preferences.AppProfiles().Save(profiles); err != nil {
					dialog.ShowError(err, p.window)
				} else {
					dialog.ShowInformation("Remove Profile", fmt.Sprintf("Remove %q", profile.Name), p.window)
				}
				p.profileSelector.SetSelectedIndex(0)
			},
			window,
		).Show()
	})
	removeItem.Icon = theme.ContentRemoveIcon()

	menu := widget.NewPopUpMenu(
		fyne.NewMenu("", editItem, removeItem),
		window.Canvas(),
	)

	serverMenuButton := widget.NewButtonWithIcon("", theme.MoreHorizontalIcon(), func() {})
	serverMenuButton.OnTapped = func() {
		menuSize := menu.Size()
		buttonSize := serverMenuButton.Size()

		position := fyne.NewPos(-(menuSize.Width - buttonSize.Width), buttonSize.Height+5)
		menu.ShowAtRelativePosition(position, serverMenuButton)
	}

	// === Pages === //
	heading := canvas.NewText("Servers", theme.Color(theme.ColorNameForeground))
	heading.TextSize = 16

	saveButton := widget.NewButton("Save Profile", func() {
		profile := p.SelectedProfile()
		if profile == nil {
			return
		}

		clientConfig := clientSettings.GetOptional()
		if clientConfig.IsEmpty() {
			profile.Client = nil
		} else {
			profile.Client = &clientConfig
		}

		if p.patcherFunc != nil && profile.Server.Patcher != nil {
			profile.Server.Patcher.Environment = p.patcherFunc()
		}

		bootConfig := profile.Server.BootConfig()
		if err := p.SaveProfile(profile, &bootConfig); err != nil {
			dialog.ShowError(err, p.window)
		} else {
			dialog.ShowInformation("User Settings", "Saved!", p.window)
		}
	})
	saveButton.Importance = widget.HighImportance
	saveButton.Icon = theme.DocumentSaveIcon()

	p.listContainer = container.NewBorder(nil, container.NewBorder(nil, nil, nil, saveButton), nil, nil,
		container.NewVScroll(
			container.NewVBox(
				heading,
				container.NewPadded(
					container.NewVBox(
						container.NewBorder(nil, nil, addServerButton, nil),
						container.NewBorder(nil, nil, nil, serverMenuButton, p.profileSelector),
					),
				),
				widget.NewSeparator(),
				p.patcherContainer,
				widget.NewAccordion(clientSettingsOverride),
			),
		),
	)

	preferences.AppProfiles().Binding().AddListener(p)

	p.ShowProfileList()
	p.Container = container.NewPadded(p.content)

	return p
}

func (p profileSettingsWidget) DataChanged() {
	p.profileSelector.SetOptions(p.AppProfiles().Get())
	p.profileSelector.SetSelectedIndex(p.profileSelector.SelectedIndex())
}

func (p *profileSettingsWidget) SetPatcherSettings(profile *Profile) {
	p.patcherContainer.RemoveAll()

	patcher, ok := profile.Server.GetPatcher()
	if !ok {
		p.patcherFunc = nil
		return
	}

	form, patcherFunc := patcher.UserForm()
	p.patcherContainer.Add(widget.NewForm(form...))
	p.patcherFunc = patcherFunc
}

func (p profileSettingsWidget) SelectedProfile() *Profile {
	profiles := p.AppProfiles().Get()
	if len(profiles) == 0 {
		return nil
	}

	return profiles[p.profileSelector.SelectedIndex()]
}

func (p profileSettingsWidget) ShowProfileList() {
	p.content.RemoveAll()
	p.content.Add(p.listContainer)
}

func (p profileSettingsWidget) SaveProfile(profile *Profile, bootConfig *BootConfig) error {
	bootPath := profile.Server.Boot
	if len(bootPath) == 0 { // Allows manually edited boot paths
		bootPath = profile.DefaultBootPath(p.settingsDir)
	}

	if err := profile.Server.SaveBootConfig(bootPath, bootConfig); err != nil {
		return fmt.Errorf("save profile: %v", err)
	}

	profile.ServerList.once = nil

	profiles := p.AppProfiles().Get()

	index := slices.IndexFunc(profiles, func(p *Profile) bool { return p != nil && p.Id == profile.Id })
	if index < 0 {
		profiles = append(profiles, profile)
	} else {
		profiles[index] = profile
	}

	return p.AppProfiles().Save(profiles)
}

func (p *profileSettingsWidget) ShowNewProfile() {
	form := NewProfileForm(nil, p.window)
	p.content.RemoveAll()

	back := widget.NewButtonWithIcon("Back", theme.NavigateBackIcon(), p.ShowProfileList)
	back.Importance = widget.LowImportance

	add := widget.NewButtonWithIcon("Create", theme.ContentAddIcon(), func() {
		if err := p.SaveProfile(form.Get()); err != nil {
			dialog.ShowError(err, p.window)
		} else {
			dialog.ShowInformation("Create Server", "Server created!", p.window)
			p.ShowProfileList()
		}
	})
	add.Importance = widget.HighImportance

	p.content.Add(
		container.NewBorder(
			nil, container.NewBorder(nil, nil, back, add), nil, nil,
			container.NewVScroll(form.Container),
		),
	)
}

func (p *profileSettingsWidget) ShowEditProfile() {
	form := NewProfileForm(p.SelectedProfile(), p.window)
	p.content.RemoveAll()

	back := widget.NewButtonWithIcon("Back", theme.NavigateBackIcon(), p.ShowProfileList)
	back.Importance = widget.LowImportance

	update := widget.NewButtonWithIcon("Save", theme.DocumentSaveIcon(), func() {
		if err := p.SaveProfile(form.Get()); err != nil {
			dialog.ShowError(err, p.window)
		} else {
			dialog.ShowInformation("Edit Server", "Server updated!", p.window)
		}
	})
	update.Importance = widget.HighImportance

	export := widget.NewButtonWithIcon("Export", theme.UploadIcon(), func() {
		dialog := dialog.NewFileSave(func(writer fyne.URIWriteCloser, err error) {
			if writer == nil {
				return
			}

			if err != nil {
				dialog.ShowError(fmt.Errorf("error when choosing file: %v", err), p.window)
				return
			}

			serverXml := form.GetXml()
			data, _ := xml.MarshalIndent(serverXml, "", "    ")

			if _, err := writer.Write(append([]byte(xml.Header), data...)); err != nil {
				dialog.ShowError(fmt.Errorf("cannot write server.xml: %v", err), p.window)
				return
			}

			dialog.ShowInformation("Export Complete", "Exported server configuration successfully!", p.window)
		}, p.window)

		dialog.SetFileName("server.xml")
		dialog.SetFilter(storage.NewExtensionFileFilter([]string{".xml"}))
		dialog.Show()
	})

	p.content.Add(
		container.NewBorder(
			nil, container.NewBorder(nil, nil, back, container.NewHBox(export, update)), nil, nil,
			container.NewVScroll(form.Container),
		),
	)
}

type launcherSettingsWidget struct {
	*fyne.Container
}

func newLauncherSettingsWidget(window fyne.Window, preferences Preferences) *launcherSettingsWidget {
	settings := preferences.AppSettings().Get()

	generalHeading := canvas.NewText("General", theme.Color(theme.ColorNameForeground))
	generalHeading.TextSize = 16

	closeOnPlay := widget.NewCheck("Closes the launcher before playing", func(b bool) {})
	closeOnPlay.Checked = settings.Launch.CloseOnPlay

	reviewPatches := widget.NewCheck("Display a summary of a patch before updating", func(b bool) {})
	reviewPatches.Checked = settings.Launch.ReviewPatchesBeforeUpdate

	clientHeading := canvas.NewText("Default Client", theme.Color(theme.ColorNameForeground))
	clientHeading.TextSize = 16

	clientSettings := nlwidgets.NewClientSettings(window, true, settings.Launch.DefaultClient.ToOptional())

	etcSettings, etcFunc := NewEtcSettings(window, settings)

	saveButton := widget.NewButtonWithIcon("Save Launcher", theme.DocumentSaveIcon(), func() {
		settings := Settings{
			Launch: LaunchConfig{
				DefaultClient:             clientSettings.Get(),
				CloseOnPlay:               closeOnPlay.Checked,
				ReviewPatchesBeforeUpdate: reviewPatches.Checked,
			},
		}
		settings.Launch.DefaultClient.Etc = etcFunc()

		if err := preferences.AppSettings().Save(settings); err != nil {
			slog.Error("Failed to save settings", "error", err)
			dialog.ShowError(err, window)
		} else {
			dialog.ShowInformation("Launcher Settings", "Settings saved!", window)
		}
	})
	saveButton.Importance = widget.HighImportance

	return &launcherSettingsWidget{
		container.NewBorder(
			nil, container.NewBorder(nil, nil, nil, saveButton), nil, nil,
			container.NewVScroll(
				container.NewVBox(
					generalHeading,
					widget.NewForm(
						widget.NewFormItem("Close On Play", closeOnPlay),
						widget.NewFormItem("Review Patches", reviewPatches),
					),
					widget.NewSeparator(),
					clientHeading,
					widget.NewForm(clientSettings.Form()...),
					etcSettings,
				),
			),
		),
	}
}

func NewSettingsWindow(app fyne.App, preferences Preferences, settingsDir string) fyne.Window {
	window := app.NewWindow("Settings")
	window.Resize(fyne.NewSize(800, 600))
	window.SetIcon(theme.SettingsIcon())
	window.SetFixedSize(true)

	heading := canvas.NewText("Settings", theme.Color(theme.ColorNameForeground))
	heading.TextSize = 24

	profiles := newProfileSettingsWidget(window, preferences, settingsDir)
	launcher := newLauncherSettingsWidget(window, preferences)

	window.SetContent(
		container.NewPadded(
			container.NewBorder(
				heading, nil, nil, nil,
				container.NewAppTabs(
					container.NewTabItem("Profiles", profiles.Container),
					container.NewTabItem("Launcher", launcher.Container),
				),
			),
		),
	)

	return window
}
