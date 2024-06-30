package nlwindows

import (
	"log"
	"net/url"
	"os"
	"os/exec"
	"path"
	"runtime"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/I-Am-Dench/nimbus-launcher/version"
)

func mustParse(rawUrl string) *url.URL {
	url, err := url.Parse(rawUrl)
	if err != nil {
		panic(err)
	}
	return url
}

var RepoURL = mustParse("https://github.com/I-Am-Dench/nimbus-launcher")

func OpenLicense() {
	dir := "."
	if exe, err := os.Executable(); err == nil {
		dir = path.Dir(exe)
	}

	// Only for windows -- Needs `open -t [or -e]` for Mac or equivalent for Linux
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("notepad", path.Join(dir, "LICENSE"))
	case "darwin":
		cmd = exec.Command("open", "-t", path.Join(dir, "LICENSE"))
	default:
		log.Printf("OpenLicense: unsupported GOOS: %s", runtime.GOOS)
		return
	}

	cmd.Stderr = os.Stdout

	log.Println("OpenLicense:", cmd)
	cmd.Run()
}

func NewInfoWindow(app fyne.App) fyne.Window {
	window := app.NewWindow("Info")
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
						widget.NewFormItem("Source", widget.NewHyperlink(RepoURL.String(), RepoURL)),
						widget.NewFormItem("License", widget.NewButton("GNU GPLv3", OpenLicense)),
					),
					container.NewStack(
						container.NewVBox(
							widget.NewLabel("Copyright (C) 2023 Theodore Friedrich (I-Am-Dench)"),
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
