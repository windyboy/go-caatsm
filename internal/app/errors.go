package app

import "errors"

// PermanentError indicates a failure that should not be retried.
type PermanentError struct {
	err error
}

// Error implements the error interface.
func (e *PermanentError) Error() string {
	if e == nil || e.err == nil {
		return ""
	}
	return e.err.Error()
}

// Unwrap allows errors.Unwrap/Is/As to inspect the underlying error.
func (e *PermanentError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

// Permanent wraps err to mark it as non-retriable.
func Permanent(err error) error {
	if err == nil {
		return nil
	}
	return &PermanentError{err: err}
}

// IsPermanent reports whether the error or any wrapped error is permanent.
func IsPermanent(err error) bool {
	var target *PermanentError
	return errors.As(err, &target)
}

