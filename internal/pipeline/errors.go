package pipeline

import "errors"

// PermanentError indicates a failure that should not be retried.
type PermanentError struct {
	err error
}

func (e *PermanentError) Error() string {
	if e == nil || e.err == nil {
		return ""
	}
	return e.err.Error()
}

// Unwrap exposes the underlying error for errors.Is / errors.As.
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

// IsPermanent reports whether err or any wrapped error is permanent.
func IsPermanent(err error) bool {
	var target *PermanentError
	return errors.As(err, &target)
}
