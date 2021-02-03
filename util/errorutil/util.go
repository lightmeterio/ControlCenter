// SPDX-FileCopyrightText: 2020,  Lightmeter <hello@lightmeter.io>
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package errorutil

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"os"
	"runtime"
)

func MustSucceed(err error, msg ...string) {
	if len(msg) > 1 {
		panic("please provide only message for each MustSucceed")
	}

	if err == nil {
		return
	}

	_, file, line, ok := runtime.Caller(1)

	if !ok {
		line = 0
		file = `<unknown file>`
	}

	errorMsg := fmt.Sprintf("FAILED: %s:%d, message:none, errors:\b", file, line)
	if len(msg) == 1 && msg[0] != "" {
		errorMsg = fmt.Sprintf("FAILED: %s:%d, message:\"%s\", errors:\b", file, line, msg[0])
	}

	log.Error().Msg(errorMsg)

	//nolint:errorlint
	if wrappedErr, ok := err.(*Error); ok {
		panic("\n" + wrappedErr.Chain().Error())
	}

	panic(err)
}

func Dief(verbose bool, err error, format string, values ...interface{}) {
	log.Error().Msgf(format, values...)

	if verbose {
		LogErrorf(err, "Detailed error")
	}

	os.Exit(1)
}

func LogErrorf(err error, format string, args ...interface{}) {
	v := ExpandError(err)
	log.Error().Interface("error", v).Msgf(format, args...)
}

func LogFatalf(err error, format string, args ...interface{}) {
	v := ExpandError(err)
	log.Fatal().Interface("error", v).Msgf(format, args...)
}

func ExpandError(err error) interface{} {
	//nolint:errorlint
	if e, ok := err.(*Error); ok {
		return e.Chain().JSON()
	}

	return err
}
