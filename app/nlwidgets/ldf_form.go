package nlwidgets

import (
	"fmt"
	"slices"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/I-Am-Dench/goverbuild/encoding/ldf"
)

type LdfForm struct {
	codebox   *CodeBox
	container *fyne.Container
}

func NewLdfForm() *LdfForm {
	codebox := NewCodeBox()
	codebox.ReadOnly = false

	return &LdfForm{
		codebox:   codebox,
		container: container.NewVBox(),
	}
}

func (f LdfForm) Get() ldf.Map {
	decoder := ldf.NewTextDecoder(strings.NewReader(f.codebox.Text))
	decoder.UseLax()

	m := ldf.Map{}
	decoder.Decode(&m)
	return m
}

func (f *LdfForm) Set(m ldf.Map) {
	data, _ := ldf.MarshalLines(m)
	f.codebox.SetText(string(data))
	f.updateSummary()
}

func (f LdfForm) Summary() *fyne.Container {
	return f.container
}

func (f LdfForm) updateSummary() error {
	decoder := ldf.NewTextDecoder(strings.NewReader(f.codebox.Text))
	decoder.UseLax()

	entries := []ldf.Entry{}
	if err := decoder.Decode(&entries); err != nil {
		return err
	}
	slices.SortFunc(entries, func(a, b ldf.Entry) int { return strings.Compare(a.Key, b.Key) })

	items := []*widget.FormItem{}
	for _, entry := range entries {
		var s string
		switch v := entry.Value.(type) {
		case string:
			s = v
		case []byte:
			s = string(v)
		default:
			s = fmt.Sprint(v)
		}

		items = append(items, widget.NewFormItem(entry.Key, widget.NewLabel(s)))
	}

	f.container.RemoveAll()
	f.container.Add(widget.NewForm(items...))

	return nil
}

func (f *LdfForm) ShowDialog(window fyne.Window) {
	wrapper := container.NewHScroll(f.codebox)
	wrapper.SetMinSize(fyne.NewSize(384, 256))

	dialog.ShowCustomConfirm("LDF Config", "Save", "Cancel", wrapper, func(b bool) {
		if b {
			if err := f.updateSummary(); err != nil {
				dialog.ShowError(err, window)
			}
		}
	}, window)
}
