// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package config

import (
	"errors"
	"flag"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

type fakeEnv map[string]string

func (e fakeEnv) fakeLookupenv(name string) (string, bool) {
	if value, ok := e[name]; ok {
		return value, true
	}

	return "", false
}

var noCmdline = []string{}
var noEnv = fakeEnv{}

func TestDefaultValues(t *testing.T) {
	c, err := ParseWithErrorHandling(noCmdline, noEnv.fakeLookupenv, flag.ContinueOnError)
	Convey("Incorrect default value", t, func() {
		So(c.WorkspaceDirectory, ShouldEqual, "/var/lib/lightmeter_workspace")
		So(err, ShouldBeNil)
	})
}

func TestEnvVars(t *testing.T) {
	env := fakeEnv{
		"LIGHTMETER_WORKSPACE": "/workspace",
		"LIGHTMETER_LOG_LEVEL":   "DEBUG",
	}
	c, err := ParseWithErrorHandling(noCmdline, env.fakeLookupenv, flag.ContinueOnError)
	Convey("Value could not be set using environment variable", t, func() {
		So(c.WorkspaceDirectory, ShouldEqual, "/workspace")
		So(c.LogLevel, ShouldEqual, "DEBUG")
		So(err, ShouldBeNil)
	})
}

func TestCommandLineParams(t *testing.T) {
	cmdline := []string{
		"-workspace", "/workspace",
		"-log-level", "DEBUG",
	}
	c, err := ParseWithErrorHandling(cmdline, noEnv.fakeLookupenv, flag.ContinueOnError)
	Convey("Value could not be set using command-line parameter", t, func() {
		So(c.WorkspaceDirectory, ShouldEqual, "/workspace")
		So(c.LogLevel, ShouldEqual, "DEBUG")
		So(err, ShouldBeNil)
	})
}

func TestVariablesShouldNeverOverwriteCommandLine(t *testing.T) {
	cmdline := []string{
		"-workspace", "/workspace-from-cmdline",
		"-log-level", "DEBUG",
		"-log_starting_year", "2018",
	}

	env := fakeEnv{
		"LIGHTMETER_WORKSPACE":          "/ws-from-env",
		"LIGHTMETER_LISTEN":             "localhost:9999",
		"LIGHTMETER_LOGS_STARTING_YEAR": "2020",
	}

	c, err := ParseWithErrorHandling(cmdline, env.fakeLookupenv, flag.ContinueOnError)
	Convey("Value could not be set using command-line parameter", t, func() {
		So(c.WorkspaceDirectory, ShouldEqual, "/workspace-from-cmdline")
		So(c.Address, ShouldEqual, "localhost:9999")
		So(c.LogYear, ShouldEqual, 2018)
		So(c.LogLevel, ShouldEqual, "DEBUG")
		So(err, ShouldBeNil)
	})
}

func TestHelpOption(t *testing.T) {
	_, err := ParseWithErrorHandling([]string{"-help"}, noEnv.fakeLookupenv, flag.ContinueOnError)

	Convey("Calling -help should exit with help code", t, func() {
		So(errors.Is(err, flag.ErrHelp), ShouldBeTrue)
	})
}

func TestLogPatterns(t *testing.T) {
	Convey("When not passed, get an empty array", t, func() {
		c, err := ParseWithErrorHandling(noCmdline, noEnv.fakeLookupenv, flag.ContinueOnError)
		So(err, ShouldBeNil)
		So(c.LogPatterns, ShouldResemble, []string{})
	})

	Convey("Obtain from command line", t, func() {
		c, err := ParseWithErrorHandling([]string{"-log_file_patterns", "mail.log:mail.err"}, noEnv.fakeLookupenv, flag.ContinueOnError)
		So(err, ShouldBeNil)
		So(c.LogPatterns, ShouldResemble, []string{"mail.log", "mail.err"})
	})

	Convey("Obtain from environment", t, func() {
		env := fakeEnv{"LIGHTMETER_LOG_FILE_PATTERNS": "maillog"}
		c, err := ParseWithErrorHandling([]string{"-workspace", "/lalala"}, env.fakeLookupenv, flag.ContinueOnError)
		So(err, ShouldBeNil)
		So(c.LogPatterns, ShouldResemble, []string{"maillog"})
	})

}
