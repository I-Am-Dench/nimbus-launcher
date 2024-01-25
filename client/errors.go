package client

import "fmt"

type ResourceError struct {
	Msg string
	Err error
}

func (err *ResourceError) Error() string {
	return fmt.Sprintf("%s: %v", err.Msg, err.Err)
}

func (err *ResourceError) Unwrap() error {
	return err.Err
}
