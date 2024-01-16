package forms

import (
	"io"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/I-Am-Dench/lu-launcher/ldf"
	"github.com/I-Am-Dench/lu-launcher/luwidgets"
)

type BootForm struct {
	container *fyne.Container

	serverName   *widget.Entry
	authServerIP *widget.Entry

	ugcUse3DServices *widget.Check
	locale           *widget.Select

	patchServerIP   *widget.Entry
	patchServerPort *luwidgets.IntegerEntry

	logging      *luwidgets.IntegerEntry
	dataCenterID *luwidgets.IntegerEntry
	cpCode       *luwidgets.IntegerEntry

	akamaiDLM *widget.Check

	patchServerDir *widget.Entry
	ugcServerIP    *widget.Entry
	ugcServerDir   *widget.Entry

	passURL     *widget.Entry
	signinURL   *widget.Entry
	signupURL   *widget.Entry
	registerURL *widget.Entry
	crashLogURL *widget.Entry

	trackDiskUsage *widget.Check
}

func NewBootForm(window fyne.Window) *BootForm {
	form := new(BootForm)

	bootFile := widget.NewLabel("")
	bootFile.Truncation = fyne.TextTruncateEllipsis

	bootFileOpen := widget.NewButtonWithIcon(
		"", theme.FileIcon(),
		func() {
			dialog := dialog.NewFileOpen(func(uc fyne.URIReadCloser, err error) {
				if err != nil {
					dialog.ShowError(err, window)
					return
				}

				if uc == nil || uc.URI() == nil {
					return
				}
				bootFile.SetText(uc.URI().Path())

				data, err := io.ReadAll(uc)
				if err != nil {
					dialog.ShowError(err, window)
					return
				}

				bootConfig := ldf.BootConfig{}
				err = ldf.Unmarshal(data, &bootConfig)
				if err != nil {
					dialog.ShowError(err, window)
					return
				}

				form.UpdateWith(&bootConfig)
			}, window)
			dialog.SetFilter(storage.NewExtensionFileFilter([]string{".cfg"}))

			dialog.Show()
		},
	)

	form.serverName = widget.NewEntry()
	form.serverName.PlaceHolder = "Overbuild Universe (US)"

	form.authServerIP = widget.NewEntry()
	form.authServerIP.PlaceHolder = "127.0.0.1"

	form.ugcUse3DServices = widget.NewCheck("", func(b bool) {})

	form.locale = widget.NewSelect(
		[]string{"en_US", "en_GB", "de_DE"},
		func(s string) {},
	)

	form.patchServerIP = widget.NewEntry()
	form.patchServerIP.PlaceHolder = "127.0.0.1"

	form.patchServerPort = luwidgets.NewIntegerEntry()
	form.patchServerPort.PlaceHolder = "80"

	form.logging = luwidgets.NewIntegerEntry()
	form.dataCenterID = luwidgets.NewIntegerEntry()
	form.cpCode = luwidgets.NewIntegerEntry()

	form.akamaiDLM = widget.NewCheck("", func(b bool) {})
	form.patchServerDir = widget.NewEntry()

	form.ugcServerIP = widget.NewEntry()
	form.ugcServerDir = widget.NewEntry()

	form.passURL = widget.NewEntry()
	form.signinURL = widget.NewEntry()
	form.signupURL = widget.NewEntry()
	form.registerURL = widget.NewEntry()
	form.crashLogURL = widget.NewEntry()

	form.trackDiskUsage = widget.NewCheck("", func(b bool) {})

	form.container = container.NewVBox(
		widget.NewForm(
			widget.NewFormItem("Boot File", container.NewBorder(nil, nil, bootFileOpen, nil, bootFile)),
			widget.NewFormItem("Server Name", form.serverName),
			widget.NewFormItem("Auth Server IP", form.authServerIP),
			widget.NewFormItem("UGC Use 3D Services", form.ugcUse3DServices),
			widget.NewFormItem("Locale", form.locale),
		),
		container.NewPadded(
			widget.NewAccordion(
				widget.NewAccordionItem(
					"Advanced",
					container.NewPadded(
						widget.NewForm(
							widget.NewFormItem("Patch Server IP", form.patchServerIP),
							widget.NewFormItem("Patch Server Port", form.patchServerPort),
							widget.NewFormItem("Logging", form.logging),
							widget.NewFormItem("Data Center ID", form.dataCenterID),
							widget.NewFormItem("CP Code", form.cpCode),
							widget.NewFormItem("Akamai DLM", form.akamaiDLM),
							widget.NewFormItem("Patch Server Dir", form.patchServerDir),
							widget.NewFormItem("UGC Server IP", form.ugcServerIP),
							widget.NewFormItem("UGC Server Dir", form.ugcServerDir),
							widget.NewFormItem("Password URL", form.passURL),
							widget.NewFormItem("Signin URL", form.signinURL),
							widget.NewFormItem("Signup URL", form.signupURL),
							widget.NewFormItem("Register URL", form.registerURL),
							widget.NewFormItem("Crash Log URL", form.crashLogURL),
							widget.NewFormItem("Track Disk Usage", form.trackDiskUsage),
						),
					),
				),
			),
		),
	)

	form.UpdateWith(ldf.DefaultBootConfig())

	return form
}

func (form *BootForm) UpdateWith(config *ldf.BootConfig) {
	if config == nil {
		return
	}

	form.serverName.SetText(config.ServerName)
	form.authServerIP.SetText(config.AuthServerIP)
	form.ugcUse3DServices.SetChecked(config.UGCUse3DServices)
	form.locale.SetSelected(config.Locale)

	form.patchServerIP.SetText(config.PatchServerIP)
	form.patchServerPort.SetValue(int64(config.PatchServerPort))
	form.logging.SetValue(int64(config.Logging))
	form.dataCenterID.SetValue(int64(config.DataCenterID))
	form.cpCode.SetValue(int64(config.CPCode))
	form.akamaiDLM.SetChecked(config.AkamaiDLM)
	form.patchServerDir.SetText(config.PatchServerDir)
	form.ugcServerIP.SetText(config.UGCServerIP)
	form.ugcServerDir.SetText(config.UGCServerDir)
	form.passURL.SetText(config.PasswordURL)
	form.signinURL.SetText(config.SigninURL)
	form.signupURL.SetText(config.SignupURL)
	form.registerURL.SetText(config.RegisterURL)
	form.crashLogURL.SetText(config.CrashLogURL)
	form.trackDiskUsage.SetChecked(config.TrackDiskUsage)
}

func (form *BootForm) GetConfig() *ldf.BootConfig {
	config := &ldf.BootConfig{}

	config.ServerName = form.serverName.Text
	config.PatchServerIP = form.patchServerIP.Text
	config.AuthServerIP = form.authServerIP.Text
	config.PatchServerPort = int(form.patchServerPort.Value())
	config.Logging = int(form.logging.Value())
	config.DataCenterID = uint(form.dataCenterID.Value())
	config.CPCode = int(form.cpCode.Value())
	config.AkamaiDLM = form.akamaiDLM.Checked
	config.PatchServerDir = form.patchServerDir.Text
	config.UGCUse3DServices = form.ugcUse3DServices.Checked
	config.UGCServerIP = form.ugcServerIP.Text
	config.UGCServerDir = form.ugcServerDir.Text
	config.PasswordURL = form.passURL.Text
	config.SigninURL = form.signinURL.Text
	config.SignupURL = form.signupURL.Text
	config.RegisterURL = form.registerURL.Text
	config.CrashLogURL = form.crashLogURL.Text
	config.Locale = form.locale.Selected
	config.TrackDiskUsage = form.trackDiskUsage.Checked

	return config
}

func (form *BootForm) Container() *fyne.Container {
	return form.container
}
