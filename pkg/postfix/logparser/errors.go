package parser

import (
	"errors"

	"gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser/rawparser"
)

var (
	ErrInvalidHeaderLine  = rawparser.ErrInvalidHeaderLine
	ErrUnsupportedLogLine = rawparser.ErrUnsupportedLogLine
)

func IsRecoverableError(err error) bool {
	return err == nil || errors.Is(err, ErrUnsupportedLogLine)
}
