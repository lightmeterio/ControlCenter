package parser

import (
	"gitlab.com/lightmeter/postfix-log-parser/rawparser"
)

var (
	ErrInvalidHeaderLine  = rawparser.ErrInvalidHeaderLine
	ErrUnsupportedLogLine = rawparser.ErrUnsupportedLogLine
)

func IsRecoverableError(err error) bool {
	return err == nil || err == ErrUnsupportedLogLine
}
