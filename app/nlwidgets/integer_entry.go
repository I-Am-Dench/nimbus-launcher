package nlwidgets

import (
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// Modified from: https://developer.fyne.io/extend/numerical-entry

type IntegerEntry struct {
	widget.Entry
}

func NewIntegerEntry(initial ...int64) *IntegerEntry {
	entry := &IntegerEntry{}
	entry.ExtendBaseWidget(entry)

	if len(initial) > 0 {
		entry.Text = strconv.FormatInt(initial[0], 10)
	}

	entry.PlaceHolder = "#"

	return entry
}

func (e *IntegerEntry) TypedRune(r rune) {
	if '0' <= r && r <= '9' {
		e.Entry.TypedRune(r)
	}
}

func (e *IntegerEntry) TypedShortcut(shortcut fyne.Shortcut) {
	paste, ok := shortcut.(*fyne.ShortcutPaste)
	if !ok {
		e.Entry.TypedShortcut(shortcut)
		return
	}

	content := paste.Clipboard.Content()
	if _, err := strconv.ParseInt(content, 10, 64); err == nil {
		e.Entry.TypedShortcut(shortcut)
	}
}

func (e *IntegerEntry) Value() int64 {
	i, _ := strconv.ParseInt(e.Text, 10, 64)
	return i
}

func (e *IntegerEntry) SetValue(i int64) {
	e.SetText(strconv.FormatInt(i, 10))
}
