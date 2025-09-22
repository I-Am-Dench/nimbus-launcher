package app

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/I-Am-Dench/goverbuild/models/boot"
	"github.com/I-Am-Dench/nimbus-launcher/app/nlwidgets"
)

type BootForm struct {
	Container *fyne.Container

	serverName       *widget.Entry
	authServerIp     *widget.Entry
	ugcUse3dServices *widget.Check
	locale           *nlwidgets.ItemSelector[string]
	patchServerIp    *widget.Entry
	patchServerPort  *nlwidgets.IntegerEntry
	logging          *nlwidgets.IntegerEntry
	dataCenterId     *nlwidgets.IntegerEntry
	cpCode           *nlwidgets.IntegerEntry
	akamaiDLM        *widget.Check
	patchServerDir   *widget.Entry
	ugcServerIp      *widget.Entry
	ugcServerDir     *widget.Entry
	passUrl          *widget.Entry
	signinUrl        *widget.Entry
	signupUrl        *widget.Entry
	registerUrl      *widget.Entry
	crashLogUrl      *widget.Entry
	manifestFile     *widget.Entry
	trackDiskUsage   *widget.Check
}

func (b *BootForm) Set(config boot.Config) {
	b.serverName.SetText(config.ServerName)
	b.authServerIp.SetText(config.AuthServerIP)
	b.ugcUse3dServices.SetChecked(config.UGCUse3dServices)
	b.locale.SetSelected(config.Locale)

	b.patchServerIp.SetText(config.PatchServerIP)
	b.patchServerPort.SetValue(int64(config.PatchServerPort))
	b.logging.SetValue(int64(config.Logging))
	b.dataCenterId.SetValue(int64(config.DataCenterID))
	b.cpCode.SetValue(int64(config.CPCode))
	b.akamaiDLM.SetChecked(config.AkamaiDLM)
	b.patchServerDir.SetText(config.PatchServerDir)
	b.ugcServerIp.SetText(config.UGCServerIP)
	b.ugcServerDir.SetText(config.UGCServerDir)
	b.passUrl.SetText(config.PasswordURL)
	b.signinUrl.SetText(config.SigninURL)
	b.signupUrl.SetText(config.SignupURL)
	b.registerUrl.SetText(config.RegisterURL)
	b.crashLogUrl.SetText(config.CrashLogURL)
	b.manifestFile.SetText(config.ManifestFile)
	b.trackDiskUsage.SetChecked(config.TrackDiskUsage)
}

func (b *BootForm) Get() *boot.Config {
	return &boot.Config{
		ServerName:       b.serverName.Text,
		PatchServerIP:    b.patchServerIp.Text,
		AuthServerIP:     b.authServerIp.Text,
		PatchServerPort:  int32(b.patchServerPort.Value()),
		Logging:          int32(b.logging.Value()),
		DataCenterID:     uint32(b.dataCenterId.Value()),
		CPCode:           int32(b.cpCode.Value()),
		AkamaiDLM:        b.akamaiDLM.Checked,
		PatchServerDir:   b.patchServerDir.Text,
		UGCUse3dServices: b.ugcUse3dServices.Checked,
		UGCServerIP:      b.ugcServerIp.Text,
		UGCServerDir:     b.ugcServerDir.Text,
		PasswordURL:      b.passUrl.Text,
		SigninURL:        b.signinUrl.Text,
		SignupURL:        b.signupUrl.Text,
		RegisterURL:      b.registerUrl.Text,
		CrashLogURL:      b.crashLogUrl.Text,
		Locale:           b.locale.Selected,
		ManifestFile:     b.manifestFile.Text,
		TrackDiskUsage:   b.trackDiskUsage.Checked,
	}
}

func NewBootForm() *BootForm {
	form := &BootForm{
		serverName:       widget.NewEntry(),
		authServerIp:     widget.NewEntry(),
		ugcUse3dServices: widget.NewCheck("", func(b bool) {}),
		locale:           nlwidgets.NewLocaleSelector(""),
		patchServerIp:    widget.NewEntry(),
		patchServerPort:  nlwidgets.NewIntegerEntry(),
		logging:          nlwidgets.NewIntegerEntry(),
		dataCenterId:     nlwidgets.NewIntegerEntry(),
		cpCode:           nlwidgets.NewIntegerEntry(),
		akamaiDLM:        widget.NewCheck("", func(b bool) {}),
		patchServerDir:   widget.NewEntry(),
		ugcServerIp:      widget.NewEntry(),
		ugcServerDir:     widget.NewEntry(),
		passUrl:          widget.NewEntry(),
		signinUrl:        widget.NewEntry(),
		signupUrl:        widget.NewEntry(),
		registerUrl:      widget.NewEntry(),
		crashLogUrl:      widget.NewEntry(),
		manifestFile:     widget.NewEntry(),
		trackDiskUsage:   widget.NewCheck("", func(b bool) {}),
	}

	form.serverName.PlaceHolder = "Overbuild Universe (US)"
	form.authServerIp.PlaceHolder = "127.0.0.1"
	form.patchServerIp.PlaceHolder = "127.0.0.1"
	form.patchServerPort.PlaceHolder = "80"
	form.patchServerDir.PlaceHolder = "luclient"
	form.ugcServerIp.PlaceHolder = "127.0.0.1"
	form.ugcServerDir.PlaceHolder = "3dservices"
	form.manifestFile.PlaceHolder = "trunk.txt"

	form.Container = container.NewVBox(
		widget.NewForm(
			widget.NewFormItem("Server Name", form.serverName),
			widget.NewFormItem("Auth Server IP", form.authServerIp),
			widget.NewFormItem("UGC Use 3D Services", form.ugcUse3dServices),
			widget.NewFormItem("Locale", form.locale),
		),
		widget.NewAccordion(
			widget.NewAccordionItem(
				"Advanced",
				container.NewPadded(
					widget.NewForm(
						widget.NewFormItem("Patch Server IP", form.patchServerIp),
						widget.NewFormItem("Patch Server Dir", form.patchServerDir),
						widget.NewFormItem("Logging", form.logging),
						widget.NewFormItem("Data Center ID", form.dataCenterId),
						widget.NewFormItem("CP Code", form.cpCode),
						widget.NewFormItem("Akamai DLM", form.akamaiDLM),
						widget.NewFormItem("Patch Server Dir", form.patchServerDir),
						widget.NewFormItem("UGC Server IP", form.ugcServerIp),
						widget.NewFormItem("UGC Server Dir", form.ugcServerDir),
						widget.NewFormItem("Password URL", form.passUrl),
						widget.NewFormItem("Signin URL", form.signinUrl),
						widget.NewFormItem("Signup URL", form.signupUrl),
						widget.NewFormItem("Register URL", form.registerUrl),
						widget.NewFormItem("Crash Log URL", form.crashLogUrl),
						widget.NewFormItem("Track Disk Usage", form.trackDiskUsage),
						widget.NewFormItem("Manifest File", form.manifestFile),
					),
				),
			),
		),
	)

	return form
}
