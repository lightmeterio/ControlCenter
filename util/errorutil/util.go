package errorutil

import (
	"log"
	"os"
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

func Die(verbose bool, err error, msg ...interface{}) {
	expandError := func(err error) error {
		if e, ok := err.(*Error); ok {
			return e.Chain()
		}

		return err
	}

	log.Println(msg...)

	if verbose {
		log.Println("Detailed Error:\n", expandError(err).Error())
	}

	os.Exit(1)
}
