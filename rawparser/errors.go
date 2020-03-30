package rawparser

import "errors"

var (
	InvalidHeaderLineError  = errors.New("Could not parse header")
	UnsupportedLogLineError = errors.New("Unsupported payload")
)
