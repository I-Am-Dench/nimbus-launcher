package nlwidgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

type CodeBox struct {
	widget.Entry
	ReadOnly bool
}

func NewCodeBox() *CodeBox {
	entry := &CodeBox{}
	entry.ExtendBaseWidget(entry)

	entry.MultiLine = true
	entry.Wrapping = fyne.TextTruncate
	entry.TextStyle.Monospace = true
	entry.ReadOnly = true

	return entry
}

func (b *CodeBox) TypedRune(r rune) {
	if !b.ReadOnly {
		b.Entry.TypedRune(r)
	}
}

func (b *CodeBox) TypedKey(k *fyne.KeyEvent) {
	if !b.ReadOnly {
		b.Entry.TypedKey(k)
	}
}

func (b *CodeBox) TypedShortcut(s fyne.Shortcut) {
	if !b.ReadOnly {
		b.Entry.TypedShortcut(s)
		return
	}

	switch s.(type) {
	case *fyne.ShortcutCopy, *fyne.ShortcutSelectAll:
		b.Entry.TypedShortcut(s)
	}
}
