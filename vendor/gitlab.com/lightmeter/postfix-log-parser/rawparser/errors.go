package rawparser

import "errors"

var (
	ErrInvalidHeaderLine  = errors.New("Could not parse header")
	ErrUnsupportedLogLine = errors.New("Unsupported payload")
)
