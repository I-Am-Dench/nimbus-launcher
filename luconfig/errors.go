package luconfig

import (
	"fmt"
)

type UnmarshalError struct {
	Line string
	Err  error
}

func newUnmarshalError(line string, err error) UnmarshalError {
	return UnmarshalError{
		Line: line,
		Err:  err,
	}
}

func (e UnmarshalError) Error() string {
	return fmt.Sprintf("unmarshal error at %s: %v", e.Line, e.Err)
}

func (e *UnmarshalError) Unwrap() error {
	return e.Err
}

type MarshalError struct {
	Err error
}

func newMarshalError(err error) MarshalError {
	return MarshalError{
		Err: err,
	}
}

func (e MarshalError) Error() string {
	return fmt.Sprintf("marshal error: %v", e.Err)
}

func (e *MarshalError) Unwrap() error {
	return e.Err
}
