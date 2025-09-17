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
		trackDiskUsage:   widget.NewCheck("", func(b bool) {}),
	}

	form.serverName.PlaceHolder = "Overbuild Universe (US)"
	form.authServerIp.PlaceHolder = "127.0.0.1"
	form.patchServerIp.PlaceHolder = "127.0.0.1"
	form.patchServerPort.PlaceHolder = "80"
	form.patchServerDir.PlaceHolder = "luclient"
	form.ugcServerIp.PlaceHolder = "127.0.0.1"
	form.ugcServerDir.PlaceHolder = "3dservices"

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
					),
				),
			),
		),
	)

	return form
}

// func NewBootForm(initial boot.Config) (*fyne.Container, func() *boot.Config) {
// 	serverName := widget.NewEntry()
// 	serverName.PlaceHolder = "Overbuild Universe (US)"

// 	authServerIp := widget.NewEntry()
// 	authServerIp.PlaceHolder = "127.0.0.1"

// 	ugcUse3dServices := widget.NewCheck("", func(b bool) {})

// 	locale := nlwidgets.NewLocaleSelector("")

// 	patchServerIp := widget.NewEntry()
// 	patchServerIp.PlaceHolder = "127.0.0.1"

// 	patchServerPort := nlwidgets.NewIntegerEntry()
// 	patchServerPort.PlaceHolder = "80"

// 	logging := nlwidgets.NewIntegerEntry()
// 	dataCenterId := nlwidgets.NewIntegerEntry()
// 	cpCode := nlwidgets.NewIntegerEntry()

// 	akamaiDLM := widget.NewCheck("", func(b bool) {})

// 	patchServerDir := widget.NewEntry()
// 	patchServerDir.PlaceHolder = "luclient"

// 	ugcServerIp := widget.NewEntry()
// 	ugcServerIp.PlaceHolder = "127.0.0.1"

// 	ugcServerDir := widget.NewEntry()
// 	ugcServerDir.PlaceHolder = "3dservices"

// 	passUrl := widget.NewEntry()
// 	signinUrl := widget.NewEntry()
// 	signupUrl := widget.NewEntry()
// 	registerUrl := widget.NewEntry()
// 	crashLogUrl := widget.NewEntry()

// 	trackDiskUsage := widget.NewCheck("", func(b bool) {})

// 	serverName.SetText(initial.ServerName)
// 	authServerIp.SetText(initial.AuthServerIP)
// 	ugcUse3dServices.SetChecked(initial.UGCUse3dServices)
// 	locale.SetSelected(initial.Locale)

// 	patchServerIp.SetText(initial.PatchServerIP)
// 	patchServerPort.SetValue(int64(initial.PatchServerPort))
// 	logging.SetValue(int64(initial.Logging))
// 	dataCenterId.SetValue(int64(initial.DataCenterID))
// 	cpCode.SetValue(int64(initial.CPCode))
// 	akamaiDLM.SetChecked(initial.AkamaiDLM)
// 	patchServerDir.SetText(initial.PatchServerDir)
// 	ugcServerIp.SetText(initial.UGCServerIP)
// 	ugcServerDir.SetText(initial.UGCServerDir)
// 	passUrl.SetText(initial.PasswordURL)
// 	signinUrl.SetText(initial.SigninURL)
// 	signupUrl.SetText(initial.SignupURL)
// 	registerUrl.SetText(initial.RegisterURL)
// 	crashLogUrl.SetText(initial.CrashLogURL)
// 	trackDiskUsage.SetChecked(initial.TrackDiskUsage)

// 	get := func() *boot.Config {
// 		return &boot.Config{
// 			ServerName:       serverName.Text,
// 			PatchServerIP:    patchServerIp.Text,
// 			AuthServerIP:     authServerIp.Text,
// 			PatchServerPort:  int32(patchServerPort.Value()),
// 			Logging:          int32(logging.Value()),
// 			DataCenterID:     uint32(dataCenterId.Value()),
// 			CPCode:           int32(cpCode.Value()),
// 			AkamaiDLM:        akamaiDLM.Checked,
// 			PatchServerDir:   patchServerDir.Text,
// 			UGCUse3dServices: ugcUse3dServices.Checked,
// 			UGCServerIP:      ugcServerIp.Text,
// 			UGCServerDir:     ugcServerDir.Text,
// 			PasswordURL:      passUrl.Text,
// 			SigninURL:        signinUrl.Text,
// 			SignupURL:        signupUrl.Text,
// 			RegisterURL:      registerUrl.Text,
// 			CrashLogURL:      crashLogUrl.Text,
// 			Locale:           locale.Selected,
// 			TrackDiskUsage:   trackDiskUsage.Checked,
// 		}
// 	}

// 	return container.NewVBox(
// 		widget.NewForm(
// 			widget.NewFormItem("Server Name", serverName),
// 			widget.NewFormItem("Auth Server IP", authServerIp),
// 			widget.NewFormItem("UGC Use 3D Services", ugcUse3dServices),
// 			widget.NewFormItem("Locale", locale),
// 		),
// 		widget.NewAccordion(
// 			widget.NewAccordionItem(
// 				"Advanced",
// 				container.NewPadded(
// 					widget.NewForm(
// 						widget.NewFormItem("Patch Server IP", patchServerIp),
// 						widget.NewFormItem("Patch Server Dir", patchServerDir),
// 						widget.NewFormItem("Logging", logging),
// 						widget.NewFormItem("Data Center ID", dataCenterId),
// 						widget.NewFormItem("CP Code", cpCode),
// 						widget.NewFormItem("Akamai DLM", akamaiDLM),
// 						widget.NewFormItem("Patch Server Dir", patchServerDir),
// 						widget.NewFormItem("UGC Server IP", ugcServerIp),
// 						widget.NewFormItem("UGC Server Dir", ugcServerDir),
// 						widget.NewFormItem("Password URL", passUrl),
// 						widget.NewFormItem("Signin URL", signinUrl),
// 						widget.NewFormItem("Signup URL", signupUrl),
// 						widget.NewFormItem("Register URL", registerUrl),
// 						widget.NewFormItem("Crash Log URL", crashLogUrl),
// 						widget.NewFormItem("Track Disk Usage", trackDiskUsage),
// 					),
// 				),
// 			),
// 		),
// 	), get
// }
