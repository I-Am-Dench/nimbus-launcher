package multiwindow

import "fyne.io/fyne/v2"

type window struct {
	fyne.Window

	onClosed func()
}

func (window *window) SetOnClosed(onClosed func()) {
	window.onClosed = onClosed
	window.Window.SetOnClosed(onClosed)
}

func (window *window) Hide() {
	if window.onClosed != nil {
		window.onClosed()
	}

	window.Window.Hide()
}
