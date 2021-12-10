// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package errorutil

import (
	"fmt"
	"github.com/hashicorp/go-multierror"
	"github.com/rs/zerolog/log"
	"io"
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

func Dief(err error, format string, values ...interface{}) {
	log.Error().Msgf(format, values...)
	LogErrorf(err, "Detailed error")
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

func UpdateErrorFromCall(f func() error, err *error) {
	if err == nil {
		log.Fatal().Msg("err must be not nil!")
	}

	cErr := f()

	if cErr == nil {
		return
	}

	if *err == nil {
		*err = Wrap(cErr)
		return
	}

	*err = multierror.Append(*err, Wrap(cErr))
}

// UpdateErrorFromCloser is supposed to be used to close a io.Closer when a function exits,
// merging its error return into err, if any.
func UpdateErrorFromCloser(closer io.Closer, err *error) {
	UpdateErrorFromCall(closer.Close, err)
}
