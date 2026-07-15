package weather

import "errors"

var (
	// ErrInvalidFormat indicates an invalid weather report format
	ErrInvalidFormat = errors.New("invalid weather report format")

	// ErrUnsupportedToken indicates an unsupported token in the report
	ErrUnsupportedToken = errors.New("unsupported token")

	// ErrMissingStation indicates missing station identifier
	ErrMissingStation = errors.New("missing station identifier")

	// ErrMissingTime indicates missing time information
	ErrMissingTime = errors.New("missing time information")
)

