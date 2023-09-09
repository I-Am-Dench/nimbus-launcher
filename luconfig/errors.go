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
