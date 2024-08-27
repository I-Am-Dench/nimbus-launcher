package info

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
	"github.com/I-Am-Dench/nimbus-launcher/app/multiwindow"
	"github.com/I-Am-Dench/nimbus-launcher/version"
)

func mustParse(rawUrl string) *url.URL {
	url, err := url.Parse(rawUrl)
	if err != nil {
		panic(err)
	}
	return url
}

var RepoUrl = mustParse("https://github.com/I-Am-Dench/nimbus-launcher")

func OpenLicense() {
	dir := "."
	if exe, err := os.Executable(); err == nil {
		dir = filepath.Dir(exe)
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("notepad", filepath.Join(dir, "LICENSE"))
	case "darwin":
		cmd = exec.Command("open", "-t", filepath.Join(dir, "LICENSE"))
	case "linux":
		cmd = exec.Command("xdg-open", filepath.Join(dir, "LICENSE"))
	default:
		log.Printf("OpenLicense: unsupported GOOS: %s", runtime.GOOS)
		return
	}

	cmd.Stderr = os.Stdout

	log.Print("OpenLicense: ", cmd)
	cmd.Run()
}

func New(app *multiwindow.App, id int) fyne.Window {
	window := app.NewInstanceWindow("Info", id)
	window.SetIcon(theme.InfoIcon())
	window.SetFixedSize(true)

	heading := canvas.NewText("Info", theme.ForegroundColor())
	heading.TextSize = 16

	window.SetContent(
		container.NewPadded(
			container.NewVBox(
				heading, widget.NewSeparator(),
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
