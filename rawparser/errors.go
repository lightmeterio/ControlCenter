package rawparser

import "errors"

var (
	InvalidHeaderLineError  = errors.New("Invalid Line: Could not parse header")
	UnsupportedLogLineError = errors.New("Payload not yet supported")
)
