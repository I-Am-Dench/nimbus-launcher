package app

import (
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/I-Am-Dench/nimbus-launcher/version"
)

var RepoUrl *url.URL

func init() {
	var err error
	RepoUrl, err = url.Parse("https://github.com/I-Am-Dench/nimbus-launcher")
	if err != nil {
		log.Println(err)
	}
}

func OpenLicense() {
	dir := "."
	if exe, err := os.Executable(); err == nil {
		dir = filepath.Dir(exe)
	}
	path := filepath.Join(dir, "LICENSE")

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", "-t", path)
	case "linux":
		cmd = exec.Command("xdg-open", path)
	case "windows":
		cmd = exec.Command("notepad", path)
	default:
		log.Printf("OpenLicense: unsuppored GOOS: %s", runtime.GOOS)
		return
	}

	cmd.Stderr = os.Stderr
	cmd.Run()
}

func NewInfoWindow(app fyne.App) fyne.Window {
	window := app.NewWindow("Info")
	window.SetIcon(theme.InfoIcon())
	window.SetFixedSize(true)

	heading := canvas.NewText("Info", theme.Color(theme.ColorNameForeground))
	heading.TextSize = 16

	window.SetContent(
		container.NewPadded(
			container.NewVBox(
				heading,
				widget.NewSeparator(),
				container.NewHBox(
					widget.NewForm(
						widget.NewFormItem("Version", widget.NewLabel(version.Get().Name())),
						widget.NewFormItem("Revision", widget.NewLabel(version.Revision())),
						widget.NewFormItem("Author", widget.NewLabel("I-Am-Dench")),
						widget.NewFormItem("Source", widget.NewHyperlink(RepoUrl.String(), RepoUrl)),
						widget.NewFormItem("License", widget.NewButton("GNU GPLv3", OpenLicense)),
					),
					container.NewStack(
						container.NewVBox(
							widget.NewLabel("Copyright (C) 2023 I-Am-Dench"),
							widget.NewLabel("This program is free software: you can redistribute it and/or modify\nit under the terms of the GNU General Public License as published by\nthe Free Software Foundation, either version 3 of the License, or\n(at your option) any later version."),
							widget.NewLabel("This program is distributed in the hope that it will be useful,\nbut WITHOUT ANY WARRANTY; without even the implied warranty of\nMERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the\nGNU General Public License for more details."),
						),
					),
				),
				widget.NewLabel("You should have received a copy of the GNU General Public License along with this program. If not, see <https://www.gnu.org/licenses/>."),
			),
		),
	)

	return window
}
