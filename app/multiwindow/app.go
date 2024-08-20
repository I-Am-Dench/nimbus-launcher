package multiwindow

import (
	"fmt"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
)

type windowState struct {
	window fyne.Window
	isOpen bool
}

type App struct {
	fyne.App

	windows map[int]*windowState
}

func New() *App {
	return &App{
		App: app.New(),

		windows: make(map[int]*windowState),
	}
}

func (app *App) NewInstanceWindow(title string, id int) fyne.Window {
	if _, ok := app.windows[id]; ok {
		panic(fmt.Errorf("multiwindow: app: unique window id is already in use (%d)", id))
	}

	window := app.NewWindow(title)
	window.SetCloseIntercept(func() {
		state := app.windows[id]
		state.isOpen = false
		state.window.Hide()
	})

	app.windows[id] = &windowState{
		window: window,
	}

	return window
}

func (app *App) ShowInstanceWindow(id int) {
	state, ok := app.windows[id]
	if !ok {
		log.Printf("multiwindow: app: no window with unique id %d", id)
		return
	}

	if state.isOpen {
		state.window.RequestFocus()
		return
	}

	state.isOpen = true
	state.window.CenterOnScreen()
	state.window.Show()
}
