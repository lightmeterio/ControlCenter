package errorutil

import (
	"log"
	"runtime"
)

func MustSucceed(err error, msg string) {
	if err == nil {
		return
	}
	_, file, line, ok := runtime.Caller(1)

	if !ok {
		line = 0
		file = `<unknown file>`
	}

	log.Printf("FAILED: %s:%d, message:\"%s\", errors:\b", file, line, msg)

	if wrappedErr, ok := err.(*Error); ok {
		panic("\n" + wrappedErr.Chain().Error())
	}

	panic(err)
}
