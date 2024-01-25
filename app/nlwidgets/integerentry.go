package nlwidgets

import (
	"fmt"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// Modified from: https://developer.fyne.io/extend/numerical-entry

type IntegerEntry struct {
	widget.Entry
}

func NewIntegerEntry(initial ...int64) *IntegerEntry {
	entry := new(IntegerEntry)
	entry.ExtendBaseWidget(entry)

	if len(initial) > 0 {
		entry.Text = fmt.Sprint(initial[0])
	}

	entry.PlaceHolder = "#"

	return entry
}

func (entry *IntegerEntry) TypedRune(r rune) {
	if '0' <= r && r <= '9' {
		entry.Entry.TypedRune(r)
	}
}

func (entry *IntegerEntry) TypedShortcut(shortcut fyne.Shortcut) {
	paste, ok := shortcut.(*fyne.ShortcutPaste)
	if !ok {
		entry.Entry.TypedShortcut(shortcut)
		return
	}

	content := paste.Clipboard.Content()
	if _, err := strconv.ParseInt(content, 10, 64); err == nil {
		entry.Entry.TypedShortcut(shortcut)
	}
}

func (entry *IntegerEntry) Value() int64 {
	i, _ := strconv.ParseInt(entry.Text, 10, 64)
	return i
}

func (entry *IntegerEntry) SetValue(i int64) {
	entry.SetText(fmt.Sprint(i))
}
