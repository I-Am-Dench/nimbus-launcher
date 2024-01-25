package patch

import (
	"errors"
	"fmt"
)

var (
	ErrPatchesUnsupported  = errors.New("patches unsupported on remote")
	ErrPatchesUnavailable  = errors.New("patch server could not be reached")
	ErrPatchesUnauthorized = errors.New("invalid patch token")
)

type PatchError struct {
	Err error
}

func (err *PatchError) Error() string {
	return fmt.Sprintf("patch error: %v", err.Err)
}

func (err *PatchError) Unwrap() error {
	return err.Err
}
