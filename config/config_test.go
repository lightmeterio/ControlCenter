// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package config

import (
	"encoding/json"
	"errors"
	"flag"
	"gitlab.com/lightmeter/controlcenter/metadata"
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
		So(c.Verbose, ShouldBeFalse)
		So(err, ShouldBeNil)
	})
}

func TestEnvVars(t *testing.T) {
	env := fakeEnv{
		"LIGHTMETER_WORKSPACE": "/workspace",
		"LIGHTMETER_VERBOSE":   "true",
	}
	c, err := ParseWithErrorHandling(noCmdline, env.fakeLookupenv, flag.ContinueOnError)
	Convey("Value could not be set using environment variable", t, func() {
		So(c.WorkspaceDirectory, ShouldEqual, "/workspace")
		So(c.Verbose, ShouldBeTrue)
		So(err, ShouldBeNil)
	})
}

func TestCommandLineParams(t *testing.T) {
	cmdline := []string{
		"-workspace", "/workspace",
		"-verbose", "1",
	}
	c, err := ParseWithErrorHandling(cmdline, noEnv.fakeLookupenv, flag.ContinueOnError)
	Convey("Value could not be set using command-line parameter", t, func() {
		So(c.WorkspaceDirectory, ShouldEqual, "/workspace")
		So(c.Verbose, ShouldBeTrue)
		So(err, ShouldBeNil)
	})
}

func TestVariablesShouldNeverOverwriteCommandLine(t *testing.T) {
	cmdline := []string{
		"-workspace", "/workspace-from-cmdline",
		"-verbose",
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
		So(c.Verbose, ShouldBeTrue)
		So(err, ShouldBeNil)
	})
}

func TestWrongCommandLineInputType(t *testing.T) {
	cmdline := []string{
		"-verbose=Schrödinger",
	}
	_, err := ParseWithErrorHandling(cmdline, noEnv.fakeLookupenv, flag.ContinueOnError)
	Convey("Wrong input value should raise an error", t, func() {
		So(err, ShouldNotBeNil)
	})
}

func TestWrongEnvVarInputType(t *testing.T) {
	env := fakeEnv{
		"LIGHTMETER_VERBOSE": "Schrödinger",
	}
	_, err := ParseWithErrorHandling(noCmdline, env.fakeLookupenv, flag.ContinueOnError)
	Convey("Wrong input value should raise an error", t, func() {
		So(err, ShouldNotBeNil)
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

func TestDefaultSettings(t *testing.T) {
	Convey("When not passed, get an empty map", t, func() {
		c, err := ParseWithErrorHandling(noCmdline, noEnv.fakeLookupenv, flag.ContinueOnError)
		So(err, ShouldBeNil)
		So(c.DefaultSettings, ShouldResemble, metadata.DefaultValues{})
	})

	Convey("Obtain from command line", t, func() {
		c, err := ParseWithErrorHandling([]string{"-default_settings", `{"key1": {"subkey1": 42, "subkey2": "hi"}}`}, noEnv.fakeLookupenv, flag.ContinueOnError)
		So(err, ShouldBeNil)
		So(c.DefaultSettings, ShouldResemble, metadata.DefaultValues{"key1": map[string]interface{}{"subkey1": json.Number("42"), "subkey2": "hi"}})
	})

	Convey("Obtain from environment", t, func() {
		env := fakeEnv{"LIGHTMETER_DEFAULT_SETTINGS": `{"key1": {"subkey1": 42, "subkey2": "hi"}}`}
		c, err := ParseWithErrorHandling([]string{"-workspace", "/lalala"}, env.fakeLookupenv, flag.ContinueOnError)
		So(err, ShouldBeNil)
		So(c.DefaultSettings, ShouldResemble, metadata.DefaultValues{"key1": map[string]interface{}{"subkey1": json.Number("42"), "subkey2": "hi"}})
	})

	Convey("Fail to parse default settings", t, func() {
		env := fakeEnv{"LIGHTMETER_DEFAULT_SETTINGS": `{this is not json^^56565`}
		_, err := ParseWithErrorHandling([]string{"-workspace", "/lalala"}, env.fakeLookupenv, flag.ContinueOnError)
		So(err, ShouldNotBeNil)
	})

}
