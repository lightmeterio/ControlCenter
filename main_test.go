// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"fmt"
	"os"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestEnvVars(t *testing.T) {

	type stringParam struct {
		default_value string
		setvar        *string
	}
	ParseFlags()
	stringParams := map[string]stringParam{
		"LIGHTMETER_WORKSPACE":   stringParam{"/var/lib/lightmeter_workspace", &workspaceDirectory},
		"LIGHTMETER_WATCH_DIR":   stringParam{"", &dirToWatch},
		"LIGHTMETER_LISTEN":      stringParam{":8080", &address},
		"LIGHTMETER_LOGS_SOCKET": stringParam{"", &socket},
		"LIGHTMETER_LOG_FORMAT":  stringParam{"default", &logFormat},
	}

	for envname, param := range stringParams {
		unset_string := "this value indicates that a parameter was not set"
		*param.setvar = unset_string

		// first check that the default value is correct (when param is not set via an env var)
		os.Unsetenv(envname)
		ParseFlags()
		Convey(fmt.Sprint("default value for parameter", envname, "is incorrect"), t, func() {
			So(*param.setvar, ShouldEqual, param.default_value)
		})

		// then try two arbitrary string values
		for _, val := range []string{"abcd^89", "1efgh35"} {
			*param.setvar = unset_string
			os.Setenv(envname, val)
			ParseFlags()
			Convey(fmt.Sprint("value for parameter", envname, "could not be set using an environment variable"), t, func() {
				So(*param.setvar, ShouldEqual, val)
			})
		}
	}

	// TODO: boolean variables
}
