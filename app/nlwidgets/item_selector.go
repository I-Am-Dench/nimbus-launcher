package nlwidgets

import (
	"fyne.io/fyne/v2/widget"
)

type ItemSelector[T any] struct {
	widget.Select

	options []T
	strFunc func(T) string
	cmpFunc func(T, T) bool

	OnChanged func(T)

	Selected T
}

func NewItemSelector[T any](options []T, strFunc func(T) string, cmpFunc func(T, T) bool) *ItemSelector[T] {
	s := &ItemSelector[T]{
		strFunc: strFunc,
		cmpFunc: cmpFunc,
	}
	s.ExtendBaseWidget(s)

	s.Select.OnChanged = func(string) {
		index := s.SelectedIndex()
		if index < 0 {
			var zero T
			s.Selected = zero
		} else {
			s.Selected = s.options[index]
		}

		if s.OnChanged != nil {
			s.OnChanged(s.Selected)
		}
	}

	s.SetOptions(options)

	return s
}

func (s *ItemSelector[T]) SetOptions(options []T) {
	s.options = options

	strOptions := []string{}
	for _, v := range options {
		strOptions = append(strOptions, s.strFunc(v))
	}

	prev := s.Options
	s.Select.SetOptions(strOptions)

	if len(options) < len(prev) {
		s.SetSelectedIndex(0)
	}
}

func (s *ItemSelector[T]) ClearSelected() {
	s.Select.ClearSelected()

	var zero T
	s.Selected = zero
}

func (s *ItemSelector[T]) SetSelected(a T) {
	for i, b := range s.options {
		if s.cmpFunc(a, b) {
			s.SetSelectedIndex(i)
			return
		}
	}
}
