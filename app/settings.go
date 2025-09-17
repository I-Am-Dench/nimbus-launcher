package app

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/I-Am-Dench/goverbuild/models/boot"
	"github.com/I-Am-Dench/nimbus-launcher/app/nlwidgets"
	"github.com/I-Am-Dench/nimbus-launcher/client"
)

type SettingsBinding struct {
	binding.Item[*Settings]
}

func (b *SettingsBinding) Settings() *Settings {
	s, _ := b.Get()
	return s
}

type Settings struct {
	Launch LaunchConfig `json:"launch"`
}

var DefaultSettings = Settings{
	Launch: LaunchConfig{
		DefaultClient:             client.DefaultConfig,
		CloseOnPlay:               true,
		ReviewPatchesBeforeUpdate: true,
	},
}

func compareProfiles(a, b *Profile) bool {
	return a.Id == b.Id
}

func profileName(p *Profile) string {
	return p.Name
}

type profileSettings struct {
	*fyne.Container
	ProfileListBinding
	window fyne.Window

	profilesPath string

	profileSelector  *nlwidgets.ItemSelector[*Profile]
	listContainer    *fyne.Container
	patcherContainer *fyne.Container
	patcherFunc      PatcherFunc

	content *fyne.Container
}

func newProfileSettings(window fyne.Window, profilesBinding ProfileListBinding, profilesPath string) *profileSettings {
	p := &profileSettings{
		ProfileListBinding: profilesBinding,
		window:             window,
		profilesPath:       profilesPath,
		patcherContainer:   container.NewStack(),
		content:            container.NewStack(),
	}

	directory := nlwidgets.NewDirectorySelector(window)

	clientName := widget.NewEntry()
	clientName.PlaceHolder = client.DefaultExe

	clientSettings := widget.NewAccordionItem(
		"Client (Override Default)",
		widget.NewForm(
			widget.NewFormItem("Directory", directory),
			widget.NewFormItem("Client Name", clientName),
		),
	)

	// === Selector
	p.profileSelector = nlwidgets.NewItemSelector(p.Profiles(), profileName, compareProfiles, func(profile *Profile) {
		if profile == nil {
			return
		}

		p.SetPatcherSettings(profile)

		if profile.Client == nil {
			directory.SetText("")
			clientName.SetText("")
		} else {
			directory.SetText(profile.Client.Directory)
			clientName.SetText(profile.Client.Name)
		}

		clientSettings.Open = profile.Client != nil
	})
	p.profileSelector.PlaceHolder = "(Select server)"
	p.profileSelector.SetSelectedIndex(0)

	addServerButton := widget.NewButtonWithIcon("Add Server", theme.ContentAddIcon(), p.ShowNewProfile)
	addServerButton.Importance = widget.LowImportance

	// === Menu
	editItem := fyne.NewMenuItem("Edit", p.ShowEditProfile)
	editItem.Icon = theme.DocumentCreateIcon()

	removeItem := fyne.NewMenuItem("Remove", func() {
		index := p.profileSelector.SelectedIndex()
		profiles := p.Profiles()

		profile := profiles[index]
		if profile == nil {
			return
		}

		dialog.NewCustomConfirm(
			"Remove Server",
			"Remove",
			"Cancel",
			widget.NewLabel(fmt.Sprintf("Remove profile '%s'?", profile.Name)),
			func(ok bool) {
				if !ok {
					return
				}

				profiles = append(profiles[:index], profiles[index+1:]...)
				profilesBinding.Set(profiles)

				if err := p.SaveProfiles(profiles); err != nil {
					dialog.ShowError(err, p.window)
				} else {
					dialog.ShowInformation("Remove Profile", fmt.Sprint("Removed ", profile.Name), window)
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

	// === Pages
	heading := canvas.NewText("Servers", theme.Color(theme.ColorNameForeground))
	heading.TextSize = 16

	saveButton := widget.NewButton("Save", func() {
		profile := p.SelectedProfile()
		if profile == nil {
			return
		}

		if len(directory.Text) == 0 && len(clientName.Text) == 0 {
			profile.Client = nil
		} else {
			profile.Client = &client.Config{
				Directory: directory.Text,
				Name:      clientName.Text,
			}
		}

		if p.patcherFunc != nil && profile.Server.Patcher != nil {
			profile.Server.Patcher.Environment = p.patcherFunc()
		}

		if err := p.SaveProfile(profile, profile.Server.BootConfig()); err != nil {
			dialog.ShowError(err, p.window)
		} else {
			dialog.ShowInformation("Save User Settings", "Saved!", p.window)
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
				widget.NewAccordion(clientSettings),
			),
		),
	)

	profilesBinding.AddListener(p)

	p.ShowProfileList()
	p.Container = container.NewPadded(p.content)

	return p
}

func (p *profileSettings) DataChanged() {
	p.profileSelector.SetOptions(p.Profiles())
	p.profileSelector.SetSelectedIndex(p.profileSelector.SelectedIndex())
}

func (p *profileSettings) SetPatcherSettings(profile *Profile) {
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

func (p *profileSettings) SelectedProfile() *Profile {
	profiles := p.Profiles()
	if len(profiles) == 0 {
		return nil
	}

	return profiles[p.profileSelector.SelectedIndex()]
}

func (s *profileSettings) ShowProfileList() {
	s.content.RemoveAll()
	s.content.Add(s.listContainer)
}

func (s *profileSettings) SaveProfiles(profiles []*Profile) error {
	data, err := json.MarshalIndent(profiles, "", "    ")
	if err != nil {
		return fmt.Errorf("save profiles: %v", err)
	}

	if err := os.WriteFile(s.profilesPath, data, 0664); err != nil {
		return fmt.Errorf("save profiles: %v", err)
	}

	s.Set(profiles)
	return nil
}

func (s *profileSettings) SaveProfile(profile *Profile, bootConfig *boot.Config) error {
	bootDir := filepath.Join(filepath.Dir(s.profilesPath), BootDir)
	if err := os.MkdirAll(bootDir, 0755); err != nil {
		return fmt.Errorf("save profile: %v", err)
	}

	bootPath := profile.Server.Boot
	if len(bootPath) == 0 { // Allows manually edited boot paths
		bootPath = filepath.Join(bootDir, profile.Id+".cfg")
	}

	if err := profile.Server.SaveBootConfig(bootPath, bootConfig); err != nil {
		return fmt.Errorf("save profile: %v", err)
	}

	profiles := s.Profiles()

	index := slices.IndexFunc(profiles, func(p *Profile) bool { return p != nil && p.Id == profile.Id })
	if index < 0 {
		profiles = append(profiles, profile)
	} else {
		profiles[index] = profile
	}

	return s.SaveProfiles(profiles)
}

func (s *profileSettings) ShowNewProfile() {
	form := NewProfileForm(nil, s.window)
	s.content.RemoveAll()

	back := widget.NewButtonWithIcon("Back", theme.NavigateBackIcon(), s.ShowProfileList)
	back.Importance = widget.LowImportance

	add := widget.NewButtonWithIcon("Create", theme.ContentAddIcon(), func() {
		if err := s.SaveProfile(form.Get()); err != nil {
			dialog.ShowError(err, s.window)
		} else {
			dialog.ShowInformation("Create Server", "Server created!", s.window)
			s.ShowProfileList()
		}
	})
	add.Importance = widget.HighImportance

	s.content.Add(
		container.NewBorder(
			nil, container.NewBorder(nil, nil, back, add), nil, nil,
			container.NewVScroll(form.Container),
		),
	)
}

func (s *profileSettings) ShowEditProfile() {
	form := NewProfileForm(s.SelectedProfile(), s.window)
	s.content.RemoveAll()

	back := widget.NewButtonWithIcon("Back", theme.NavigateBackIcon(), s.ShowProfileList)
	back.Importance = widget.LowImportance

	update := widget.NewButtonWithIcon("Save", theme.DocumentSaveIcon(), func() {
		if err := s.SaveProfile(form.Get()); err != nil {
			dialog.ShowError(err, s.window)
		} else {
			dialog.ShowInformation("Edit Server", "Server updated!", s.window)
		}
	})
	update.Importance = widget.HighImportance

	export := widget.NewButtonWithIcon("Export", theme.UploadIcon(), func() {
		dialog := dialog.NewFileSave(func(writer fyne.URIWriteCloser, err error) {
			if writer == nil {
				return
			}

			if err != nil {
				dialog.ShowError(fmt.Errorf("error when choosing file: %v", err), s.window)
				return
			}

			serverXml := form.GetXml()
			data, _ := xml.MarshalIndent(serverXml, "", "    ")

			if _, err := writer.Write(append([]byte(xml.Header), data...)); err != nil {
				dialog.ShowError(fmt.Errorf("cannot write server.xml: %v", err), s.window)
				return
			}

			dialog.ShowInformation("Export Complete", "Exported server configuration successfully!", s.window)
		}, s.window)

		dialog.SetFileName("server.xml")
		dialog.SetFilter(storage.NewExtensionFileFilter([]string{".xml"}))
		dialog.Show()
	})

	s.content.Add(
		container.NewBorder(
			nil, container.NewBorder(nil, nil, back, container.NewHBox(export, update)), nil, nil,
			container.NewVScroll(form.Container),
		),
	)
}

type launcherSettings struct {
	*fyne.Container
	SettingsBinding
}

func newLauncherSettings(window fyne.Window, settingsBinding SettingsBinding) *launcherSettings {
	l := &launcherSettings{
		SettingsBinding: settingsBinding,
	}

	settings := l.Settings()

	generalHeading := canvas.NewText("General", theme.Color(theme.ColorNameForeground))
	generalHeading.TextSize = 16

	closeOnPlay := widget.NewCheck("Closes the launcher before playing", func(b bool) {})
	closeOnPlay.Checked = settings.Launch.CloseOnPlay

	reviewPatches := widget.NewCheck("Display a summary of a patch before updating", func(b bool) {})
	reviewPatches.Checked = settings.Launch.ReviewPatchesBeforeUpdate

	clientHeading := canvas.NewText("Default Client", theme.Color(theme.ColorNameForeground))
	clientHeading.TextSize = 16

	installDirectory := nlwidgets.NewDirectorySelector(window)
	installDirectory.SetText(settings.Launch.DefaultClient.Directory)

	clientName := widget.NewEntry()
	clientName.PlaceHolder = "client/legouniverse.exe"
	clientName.SetText(settings.Launch.DefaultClient.Name)

	saveButton := widget.NewButtonWithIcon("Save", theme.DocumentSaveIcon(), func() {
		settings := l.Settings()

		settings.Launch.CloseOnPlay = closeOnPlay.Checked
		settings.Launch.ReviewPatchesBeforeUpdate = reviewPatches.Checked

		settings.Launch.DefaultClient = client.Config{
			Directory: installDirectory.Text,
			Name:      clientName.Text,
		}

		dialog.ShowInformation("Launcher Settings", "Settings saved!", window)
		l.Set(settings)
	})
	saveButton.Importance = widget.HighImportance

	l.Container = container.NewPadded(
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
					widget.NewForm(
						widget.NewFormItem("Directory", installDirectory),
						widget.NewFormItem("Name", clientName),
					),
				),
			),
		),
	)

	return l
}

func NewSettingsWindow(app fyne.App, settingsBinding SettingsBinding, profilesBinding ProfileListBinding, profilesPath string) fyne.Window {
	window := app.NewWindow("Settings")
	window.Resize(fyne.NewSize(800, 600))
	window.SetIcon(theme.SettingsIcon())
	window.SetFixedSize(true)

	heading := canvas.NewText("Settings", theme.Color(theme.ColorNameForeground))
	heading.TextSize = 24

	profiles := newProfileSettings(window, profilesBinding, profilesPath)
	launcher := newLauncherSettings(window, settingsBinding)

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
