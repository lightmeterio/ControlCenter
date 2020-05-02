package parser

import (
	"gitlab.com/lightmeter/postfix-log-parser/rawparser"
)

var (
	InvalidHeaderLineError  = rawparser.InvalidHeaderLineError
	UnsupportedLogLineError = rawparser.UnsupportedLogLineError
)

func IsRecoverableError(err error) bool {
	return err == nil || err == UnsupportedLogLineError
}
