package util

import (
	"log"
	"runtime"
)

func MustSucceed(err error, msg string) {
	if err != nil {
		_, file, line, ok := runtime.Caller(1)

		if !ok {
			line = 0
			file = `<unknown file>`
		}

		log.Fatal("FAILED:", file, ":", line, ` (`, msg, `), error: "`, err, `"`)
	}
}
