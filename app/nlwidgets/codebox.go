package nlwidgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

type CodeBox struct {
	widget.Entry
}

func NewCodeBox() *CodeBox {
	entry := &CodeBox{}
	entry.ExtendBaseWidget(entry)

	entry.MultiLine = true
	entry.Wrapping = fyne.TextTruncate
	entry.TextStyle.Monospace = true

	return entry
}

func (b *CodeBox) TypedRune(r rune)          {}
func (b *CodeBox) TypedKey(k *fyne.KeyEvent) {}

func (b *CodeBox) TypedShortcut(s fyne.Shortcut) {
	switch s.(type) {
	case *fyne.ShortcutCopy, *fyne.ShortcutSelectAll:
		b.Entry.TypedShortcut(s)
	}
}
