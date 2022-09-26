// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package config

import (
	"encoding/json"
	"errors"
	"flag"
	"testing"
	"time"

	"gitlab.com/lightmeter/controlcenter/metadata"

	"github.com/rs/zerolog"
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
	Convey("Default value", t, func() {
		So(c.WorkspaceDirectory, ShouldEqual, "/var/lib/lightmeter_workspace")
		So(c.LogLevel, ShouldEqual, zerolog.InfoLevel)
		So(err, ShouldBeNil)
	})
}

func TestEnvVars(t *testing.T) {
	env := fakeEnv{
		"LIGHTMETER_WORKSPACE": "/workspace",
		"LIGHTMETER_LOG_LEVEL": "DEBUG",
	}
	c, err := ParseWithErrorHandling(noCmdline, env.fakeLookupenv, flag.ContinueOnError)
	Convey("Value could not be set using environment variable", t, func() {
		So(c.WorkspaceDirectory, ShouldEqual, "/workspace")
		So(c.LogLevel, ShouldEqual, zerolog.DebugLevel)
		So(err, ShouldBeNil)
	})
}

func TestCommandLineParams(t *testing.T) {
	cmdline := []string{
		"-workspace", "/workspace",
		"-log_level", "WARN",
	}
	c, err := ParseWithErrorHandling(cmdline, noEnv.fakeLookupenv, flag.ContinueOnError)
	Convey("Value could not be set using command-line parameter", t, func() {
		So(c.WorkspaceDirectory, ShouldEqual, "/workspace")
		So(c.LogLevel, ShouldEqual, zerolog.WarnLevel)
		So(err, ShouldBeNil)
	})
}

func TestVariablesShouldNeverOverwriteCommandLine(t *testing.T) {
	cmdline := []string{
		"-workspace", "/workspace-from-cmdline",
		"-log_level", "ERROR",
		"-log_starting_year", "2018",
	}

	env := fakeEnv{
		"LIGHTMETER_WORKSPACE":          "/ws-from-env",
		"LIGHTMETER_LISTEN":             "localhost:9999",
		"LIGHTMETER_LOGS_STARTING_YEAR": "2020",
		"LIGHTMETER_LOG_LEVEL":          "DEBUG",
	}

	c, err := ParseWithErrorHandling(cmdline, env.fakeLookupenv, flag.ContinueOnError)

	Convey("Value could not be set using command-line parameter", t, func() {
		So(c.WorkspaceDirectory, ShouldEqual, "/workspace-from-cmdline")
		So(c.Address, ShouldEqual, "localhost:9999")
		So(c.LogYear, ShouldEqual, 2018)
		So(c.LogLevel, ShouldEqual, zerolog.ErrorLevel)
		So(err, ShouldBeNil)
	})
}

func TestWrongCommandLineInputType(t *testing.T) {
	cmdline := []string{
		"-log_level=Schrödinger",
	}
	_, err := ParseWithErrorHandling(cmdline, noEnv.fakeLookupenv, flag.ContinueOnError)
	Convey("Wrong input value should raise an error", t, func() {
		So(err, ShouldNotBeNil)
	})
}

func TestWrongEnvVarInputType(t *testing.T) {
	env := fakeEnv{
		"LIGHTMETER_LOG_LEVEL": "Schrödinger",
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

func TestDataRetentionDurationSettings(t *testing.T) {
	Convey("When not passed, use 3months", t, func() {
		c, err := ParseWithErrorHandling(noCmdline, noEnv.fakeLookupenv, flag.ContinueOnError)
		So(err, ShouldBeNil)
		So(c.DataRetentionDuration, ShouldEqual, time.Hour*24*90)
	})

	Convey("Obtain from command line", t, func() {
		c, err := ParseWithErrorHandling([]string{"-data_retention_duration", `105d3h`}, noEnv.fakeLookupenv, flag.ContinueOnError)
		So(err, ShouldBeNil)
		So(c.DataRetentionDuration, ShouldEqual, (time.Hour*24*105)+(time.Hour*3))
	})

	Convey("Obtain from environment", t, func() {
		env := fakeEnv{"LIGHTMETER_DATA_RETENTION_DURATION": `6w2m`}
		c, err := ParseWithErrorHandling([]string{"-workspace", "/lalala"}, env.fakeLookupenv, flag.ContinueOnError)
		So(err, ShouldBeNil)
		So(c.DataRetentionDuration, ShouldEqual, (time.Hour*24*7*6)+(time.Minute*2))
	})
}

func TestWatchDir(t *testing.T) {
	Convey("When not passed, get an empty array", t, func() {
		c, err := ParseWithErrorHandling(noCmdline, noEnv.fakeLookupenv, flag.ContinueOnError)
		So(err, ShouldBeNil)
		So(c.DirsToWatch, ShouldBeNil)
	})

	Convey("Passed one dir only via environment", t, func() {
		env := fakeEnv{"LIGHTMETER_WATCH_DIR": `/dir1`}
		c, err := ParseWithErrorHandling(noCmdline, env.fakeLookupenv, flag.ContinueOnError)
		So(err, ShouldBeNil)
		So(c.DirsToWatch, ShouldResemble, []string{"/dir1"})
	})

	Convey("Passed two dirs only via environment", t, func() {
		env := fakeEnv{"LIGHTMETER_WATCH_DIR": `/dir1:/dir2`}
		c, err := ParseWithErrorHandling(noCmdline, env.fakeLookupenv, flag.ContinueOnError)
		So(err, ShouldBeNil)
		So(c.DirsToWatch, ShouldResemble, []string{"/dir1", "/dir2"})
	})

	Convey("Passed one directory via command line", t, func() {
		c, err := ParseWithErrorHandling([]string{"-watch_dir", "/dir1"}, noEnv.fakeLookupenv, flag.ContinueOnError)
		So(err, ShouldBeNil)
		So(c.DirsToWatch, ShouldResemble, []string{"/dir1"})
	})

	Convey("Passed two directories via command line", t, func() {
		c, err := ParseWithErrorHandling([]string{"-watch_dir", "/dir1", "-watch_dir", "/dir2"}, noEnv.fakeLookupenv, flag.ContinueOnError)
		So(err, ShouldBeNil)
		So(c.DirsToWatch, ShouldResemble, []string{"/dir1", "/dir2"})
	})

	Convey("Command line overrides environment", t, func() {
		env := fakeEnv{"LIGHTMETER_WATCH_DIR": `/dir1:/dir2`}
		c, err := ParseWithErrorHandling([]string{"-watch_dir", "/dir3", "-watch_dir", "/dir4"}, env.fakeLookupenv, flag.ContinueOnError)
		So(err, ShouldBeNil)
		So(c.DirsToWatch, ShouldResemble, []string{"/dir3", "/dir4"})
	})
}
