package app

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

type CodeBox struct {
	widget.Entry
}

func NewCodeBox() *CodeBox {
	entry := new(CodeBox)
	entry.ExtendBaseWidget(entry)

	entry.MultiLine = true
	entry.Wrapping = fyne.TextTruncate
	entry.TextStyle.Monospace = true

	return entry
}

func (box *CodeBox) TypedRune(r rune)          {}
func (box *CodeBox) TypedKey(k *fyne.KeyEvent) {}

func (box *CodeBox) TypedShortcut(s fyne.Shortcut) {
	switch s.(type) {
	case *fyne.ShortcutCopy, *fyne.ShortcutSelectAll:
		box.Entry.TypedShortcut(s)
	}
}
