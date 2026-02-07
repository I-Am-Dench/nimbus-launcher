package tracker

import "github.com/I-Am-Dench/goverbuild/archive"

var _ Tracker = (*Discard)(nil)

type Discard struct{}

func (d Discard) Track(_ string, _ *archive.Archive) error { return nil }
func (d Discard) Undo() error                              { return nil }
func (d Discard) Close() error                             { return nil }
func (d Discard) GetState() State                          { return StateNormal }
func (d Discard) SetState(State)                           {}
