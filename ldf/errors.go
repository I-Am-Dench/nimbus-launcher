package ldf

import "fmt"

type UnmarshalError struct {
	Err error
}

func (err *UnmarshalError) Error() string {
	return fmt.Sprintf("unmarshal ldf: %v", err.Err)
}

func (err *UnmarshalError) Unwrap() error {
	return err.Err
}

type MarshalError struct {
	Err error
}

func (err *MarshalError) Error() string {
	return fmt.Sprintf("marshal ldf: %v", err.Err)
}

func (err *MarshalError) Unwrap() error {
	return err.Err
}
