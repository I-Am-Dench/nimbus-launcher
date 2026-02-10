package app

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strconv"
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
	"github.com/I-Am-Dench/nimbus-launcher/app/internal/defaultserver"
	"github.com/I-Am-Dench/nimbus-launcher/app/nldialogs"
	"github.com/I-Am-Dench/nimbus-launcher/app/nlwidgets"
	"github.com/I-Am-Dench/nimbus-launcher/client"
	"github.com/I-Am-Dench/nimbus-launcher/locale"
	"github.com/I-Am-Dench/nimbus-launcher/patcher"
	"github.com/I-Am-Dench/nimbus-launcher/patcher/origin"
)

const (
	BootDir = "boot"

	PreferenceSelectProfile = "selected_profile"
)

type PatcherConfig struct {
	Id          string  `json:"id" xml:"id,attr"`
	ServiceUrl  string  `json:"serviceUrl" xml:"serviceUrl"`
	Environment Patcher `json:"config,omitempty" xml:"config,omitempty"`
}

func (c *PatcherConfig) unmarshalPatcher(data []byte, unmarshaler func([]byte, any) error) error {
	if len(c.Id) == 0 {
		return nil
	}

	patcher, ok := Patchers[c.Id]
	if !ok {
		return fmt.Errorf("unknown patcher ID: %s", c.Id)
	}

	environment := patcher.Default()
	if err := unmarshaler(data, environment); err != nil {
		return err
	}

	c.Environment = environment
	return nil
}

func (c *PatcherConfig) UnmarshalJSON(data []byte) error {
	var config struct {
		Id         string          `json:"id"`
		ServiceUrl string          `json:"serviceUrl"`
		Config     json.RawMessage `json:"config"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("patcher config: %v", err)
	}

	c.Id = config.Id
	c.ServiceUrl = config.ServiceUrl

	if err := c.unmarshalPatcher(config.Config, json.Unmarshal); err != nil {
		return fmt.Errorf("patcher config: %v", err)
	}
	return nil
}

func (c *PatcherConfig) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var config struct {
		Id         string `xml:"id,attr"`
		ServiceUrl string `xml:"serviceUrl"`
		Config     struct {
			Text []byte `xml:",innerxml"`
		} `xml:"config"`
	}
	if err := d.DecodeElement(&config, &start); err != nil {
		return fmt.Errorf("patcher config: %v", err)
	}

	c.Id = config.Id
	c.ServiceUrl = config.ServiceUrl

	if err := c.unmarshalPatcher(config.Config.Text, xml.Unmarshal); err != nil {
		return fmt.Errorf("patcher config: %v", err)
	}
	return nil
}

type ServerInfo struct {
	Boot string `json:"boot,omitempty" xml:"-"`

	Account struct {
		SignIn   string `json:"signin,omitempty" xml:"signin,omitempty"`
		SignUp   string `json:"signup,omitempty" xml:"signup,omitempty"`
		Register string `json:"register,omitempty" xml:"register,omitempty"`
	} `json:"account" xml:"account"`

	Patcher *PatcherConfig `json:"patcher,omitempty" xml:"patcher,omitempty"`

	bootConfig *boot.Config `json:"-" xml:"-"`
}

func (s *ServerInfo) SaveBootConfig(path string, config *boot.Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("server info: save boot config: %v", err)
	}

	data, err := ldf.MarshalLines(config)
	if err != nil {
		return fmt.Errorf("server info: save boot config: %v", err)
	}

	if err := os.WriteFile(path, data, 0755); err != nil {
		return fmt.Errorf("server info: save boot config: %v", err)
	}
	s.Boot = path

	return nil
}

func (s *ServerInfo) LoadBootConfig() (*boot.Config, error) {
	data, err := os.ReadFile(s.Boot)
	if err != nil {
		return nil, fmt.Errorf("server info: load boot config: %v", err)
	}

	config := &boot.Config{}
	if err := ldf.UnmarshalText(data, config); err != nil {
		return nil, fmt.Errorf("server info: load boot config: %v", err)
	}

	return config, nil
}

func (s *ServerInfo) BootConfig() boot.Config {
	if s.bootConfig != nil {
		return *s.bootConfig
	}

	config, err := s.LoadBootConfig()
	if err != nil {
		slog.Error("Failed to load boot config", "error", err)
		return boot.Config{}
	}

	return *config
}

func (s *ServerInfo) SignUpUrl() string {
	if len(s.Account.SignUp) > 0 {
		return s.Account.SignUp
	}
	return s.BootConfig().SignupURL
}

func (s *ServerInfo) SignInUrl() string {
	if len(s.Account.SignIn) > 0 {
		return s.Account.SignIn
	}
	return s.BootConfig().SigninURL
}

func (s *ServerInfo) RegisterUrl() string {
	if len(s.Account.Register) > 0 {
		return s.Account.Register
	}
	return s.BootConfig().RegisterURL
}

func (s *ServerInfo) GetPatcher() (Patcher, bool) {
	if s.Patcher == nil {
		return nil, false
	}
	return s.Patcher.Environment, s.Patcher.Environment != nil
}

type Profile struct {
	Id   string `json:"id" xml:"-"`
	Name string `json:"name" xml:"name"`

	Client *client.Optional `json:"client,omitempty" xml:"client,omitempty"`
	Server ServerInfo       `json:"server" xml:"server"`

	ServerList struct {
		seq      int64
		once     func() []patcher.Server
		selected patcher.Server
	} `json:"-" xml:"-"`
}

func (p *Profile) Locale() string {
	if p.Client != nil && len(p.Client.Locale) > 0 {
		return p.Client.Locale
	}
	return p.Server.BootConfig().Locale
}

func (p *Profile) DefaultBootPath(dir string) string {
	return filepath.Join(dir, BootDir, p.Id+".cfg")
}

func (p Profile) getServerList(ctx context.Context, jar http.CookieJar) ([]patcher.Server, error) {
	patcherConfig := p.Server.Patcher
	if patcherConfig == nil || patcherConfig.Environment == nil {
		return []patcher.Server{
			defaultserver.Server{BootConfig: p.Server.BootConfig()},
		}, nil
	}

	resources, serviceUrl, err := origin.NewResources(patcherConfig.ServiceUrl)
	if err != nil {
		return nil, err
	}

	if h, ok := resources.(*origin.Http); ok {
		h.Client = &http.Client{
			Jar:       jar,
			Transport: http.DefaultTransport,
		}
	}

	slog.Info("Requesting master index", "profile", p.Name, "serviceUrl", serviceUrl)

	masterIndex, err := patcherConfig.Environment.GetMasterIndex(ctx, serviceUrl, resources)
	if err != nil {
		return nil, err
	}

	// Empty Universe Config types are assumed to be legacy netdevil patchers
	if !(masterIndex.UniverseConfig.Type == "" && patcherConfig.Id == "netdevil") && masterIndex.UniverseConfig.Type != patcherConfig.Id {
		return nil, fmt.Errorf("expected patcher \"%s\" but Master Index returned \"%s\"", patcherConfig.Id, masterIndex.UniverseConfig.Type)
	}

	if h, ok := resources.(*origin.Http); ok && len(masterIndex.Authentication) > 0 {
		resources = origin.WithAuthentication(h, nldialogs.AskForCredentials, masterIndex.Authentication)
	}

	servers, err := patcherConfig.Environment.GetServerList(ctx, resources, masterIndex)
	if err != nil {
		return nil, err
	}

	slog.Debug("Found universe config", "profile", p.Name, "numServers", len(servers))

	return servers, nil
}

func (p *Profile) ServerListOnce(window fyne.Window, ctx context.Context, jar http.CookieJar) (func() []patcher.Server, int64) {
	if p.ServerList.once == nil {
		p.ServerList.seq = time.Now().Unix()
		p.ServerList.once = sync.OnceValue(func() []patcher.Server {
			servers, err := p.getServerList(ctx, jar)
			if err != nil {
				slog.Error("Failed to fetch server list", "profile", p.Name, "error", err)
				dialog.ShowError(err, window)
				return []patcher.Server{defaultserver.Server{BootConfig: p.Server.BootConfig()}}
			}

			return servers
		})
	}

	return p.ServerList.once, p.ServerList.seq
}

func (p *Profile) SelectedServer() (patcher.Server, bool) {
	return p.ServerList.selected, p.ServerList.selected != nil
}

func DefaultBootConfig() boot.Config {
	config := boot.DefaultConfig()
	config.ManifestFile = ""
	return config
}

func DefaultProfiles(bootConfig boot.Config, profilesPath string) []*Profile {
	profile := &Profile{
		Id:   strconv.FormatInt(time.Now().Unix(), 10),
		Name: "Localhost",
	}

	if err := profile.Server.SaveBootConfig(profile.DefaultBootPath(filepath.Dir(profilesPath)), &bootConfig); err != nil {
		slog.Error(err.Error())
	}

	return []*Profile{profile}
}

func HyperLinkButton(text string, icon fyne.Resource, urlBinding binding.String) *widget.Button {
	button := widget.NewButtonWithIcon(text, icon, func() {
		rawUrl, _ := urlBinding.Get()
		if len(rawUrl) == 0 {
			return
		}

		url, err := url.Parse(rawUrl)
		if err != nil {
			slog.Error("Failed to parse URL", "url", rawUrl, "error", err)
			return
		}

		slog.Info("Opening link", "url", url)
		if err := fyne.CurrentApp().OpenURL(url); err != nil {
			slog.Error("Could not open URL", "url", url, "error", err)
		}
	})

	button.Importance = widget.LowImportance
	button.Alignment = widget.ButtonAlignLeading

	return button
}

func AddEllipsis(label *widget.Label) *widget.Label {
	label.Truncation = fyne.TextTruncateEllipsis
	return label
}

type ProfileListBinding struct {
	binding.Item[[]*Profile]
}

func (b *ProfileListBinding) Profiles() []*Profile {
	p, _ := b.Get()
	return p
}

func (b *ProfileListBinding) Options() []string {
	profiles := b.Profiles()
	if profiles == nil {
		return []string{}
	}

	options := []string{}
	for _, p := range profiles {
		options = append(options, p.Name)
	}

	return options
}

type ProfileBinding = binding.Item[*Profile]

func NewServerRadioGroup(changed func(patcher.Server)) *nlwidgets.ItemRadioGroup[patcher.Server] {
	g := nlwidgets.NewItemRadioGroup(
		[]patcher.Server{},
		func(s patcher.Server) string { return s.Info().Name },
		changed,
	)
	g.Required = true
	return g
}

type ProfileSelector struct {
	*fyne.Container
	ProfileListBinding

	window fyne.Window
	jar    http.CookieJar

	nameBinding   binding.String
	authIpBinding binding.String
	localeBinding binding.String

	signupBinding binding.String
	signinBinding binding.String

	activity *widget.Activity

	ProfileBinding ProfileBinding
	PlayingBinding binding.Bool

	statusLabel   *fyne.Container
	statusBinding StatusBinding

	serverList       *nlwidgets.ItemRadioGroup[patcher.Server]
	serverListButton *widget.Button
	serverListSeq    int64

	selector *nlwidgets.ItemSelector[*Profile]
}

func NewProfileSelector(window fyne.Window, jar http.CookieJar, profiles ProfileListBinding, onTapSettings func()) (*ProfileSelector, error) {
	s := &ProfileSelector{
		ProfileListBinding: profiles,

		window: window,
		jar:    jar,

		nameBinding:   binding.NewString(),
		authIpBinding: binding.NewString(),
		localeBinding: binding.NewString(),

		signupBinding: binding.NewString(),
		signinBinding: binding.NewString(),

		activity: widget.NewActivity(),

		ProfileBinding: binding.NewItem(func(_, _ *Profile) bool { return false }),
		PlayingBinding: binding.NewBool(),

		statusBinding: binding.NewItem(func(_, _ *patcher.Status) bool { return false }),
	}

	s.activity.Start()
	s.activity.Hide()

	s.selector = nlwidgets.NewItemSelector(s.Profiles(), profileName, compareProfiles, s.SelectServer)
	s.selector.PlaceHolder = "(Select server)"

	serverInfo := widget.NewForm(
		widget.NewFormItem(
			"Server Name", widget.NewLabelWithData(s.nameBinding),
		),
		widget.NewFormItem(
			"Auth Server IP", widget.NewLabelWithData(s.authIpBinding),
		),
		widget.NewFormItem(
			"Locale", widget.NewLabelWithData(s.localeBinding),
		),
	)

	s.serverList = NewServerRadioGroup(func(server patcher.Server) {
		profile := s.selector.Selected
		profile.ServerList.selected = server
		s.Bind(profile, server)

		s.ProfileBinding.Set(profile)

		if profile != nil && server != nil {
			fyne.CurrentApp().Preferences().SetString("profile-"+profile.Id, server.Info().Name)
		}
	})

	s.serverListButton = widget.NewButtonWithIcon("Servers", theme.ListIcon(), func() {
		list := container.NewVScroll(s.serverList)
		list.SetMinSize(list.MinSize().AddWidthHeight(128, 128))

		dialog.ShowCustom("Server List", "Ok", list, window)
	})
	s.serverListButton.Importance = widget.LowImportance

	s.statusLabel = NewStatusLabel(s.statusBinding)

	accountInfo := container.NewBorder(
		nil, nil,
		container.NewVBox(
			HyperLinkButton("Signup", theme.AccountIcon(), s.signupBinding),
			HyperLinkButton("Signin", theme.LoginIcon(), s.signinBinding),
			s.serverListButton,
		),
		nil,
		container.NewVBox(
			AddEllipsis(widget.NewLabelWithData(s.signupBinding)),
			AddEllipsis(widget.NewLabelWithData(s.signinBinding)),
			container.NewBorder(nil, nil, container.NewStack(widget.NewLabel(""), s.activity, s.statusLabel), nil),
		),
	)

	settingsButton := widget.NewButtonWithIcon("", theme.SettingsIcon(), onTapSettings)

	profiles.AddListener(s)

	s.PlayingBinding.AddListener(binding.NewDataListener(func() {
		if b, _ := s.PlayingBinding.Get(); b {
			s.selector.Disable()
		} else {
			s.selector.Enable()
		}
	}))

	s.Container = container.NewVBox(
		container.NewBorder(
			nil, nil, nil, settingsButton,
			s.selector,
		),
		container.NewGridWithColumns(2, serverInfo, accountInfo),
	)

	return s, nil
}

func (s *ProfileSelector) DataChanged() {
	s.selector.SetOptions(s.Profiles())

	if selected := fyne.CurrentApp().Preferences().String(PreferenceSelectProfile); len(selected) > 0 {
		s.selector.SetSelected(&Profile{Id: selected})
	}
}

func (s *ProfileSelector) StartLoadServers() {
	s.serverListButton.Disable()
	s.activity.Show()
	s.statusLabel.Hide()
}

func (s *ProfileSelector) StopLoadServers() {
	s.serverListButton.Enable()
	s.activity.Hide()
	s.statusLabel.Show()
}

// This can become more sophisticated later on.
// The original patcher would choose the server
// based on the selected locale and the CLOSEST
// server name match (not the exact match).
func (s ProfileSelector) findBestServer(profile *Profile, options []string) string {
	selected := fyne.CurrentApp().Preferences().String("profile-" + profile.Id)
	if slices.Contains(options, selected) {
		return selected
	}
	return options[0]
}

func (s *ProfileSelector) SetServerList(profile *Profile) {
	once, seq := profile.ServerListOnce(s.window, context.Background(), s.jar)
	s.serverListSeq = seq

	servers := once()
	if s.serverListSeq != seq {
		return
	}

	s.serverList.SetOptions(servers)

	fyne.DoAndWait(func() {
		if len(s.serverList.Options) > 0 {
			s.serverList.SetSelected(s.findBestServer(profile, s.serverList.Options))
		}
		s.StopLoadServers()
	})
}

func (s *ProfileSelector) SelectServer(profile *Profile) {
	s.Bind(nil, nil)

	if profile == nil {
		s.ProfileBinding.Set(nil)
		return
	}
	fyne.CurrentApp().Preferences().SetString(PreferenceSelectProfile, profile.Id)

	s.StartLoadServers()
	go s.SetServerList(profile)
}

func (s ProfileSelector) Bind(profile *Profile, server patcher.Server) {
	var (
		info   patcher.ServerInfo
		status *patcher.Status
	)
	if server != nil {
		info = server.Info()
		status = server.Status()
	}

	s.nameBinding.Set(info.Name)
	s.authIpBinding.Set(info.AuthIP)
	s.localeBinding.Set(locale.GetName(info.Lang))

	if profile != nil {
		bootConfig := profile.Server.BootConfig()
		s.signinBinding.Set(bootConfig.SigninURL)
		s.signupBinding.Set(bootConfig.SignupURL)
	}

	s.statusBinding.Set(status)
}
