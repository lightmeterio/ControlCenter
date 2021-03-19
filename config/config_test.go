// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package config

import (
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

var no_cmdline = []string{}
var no_env = fakeEnv{}

func TestDefaultValues(t *testing.T) {
	c, err := Parse(no_cmdline, no_env.fakeLookupenv)
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
	c, err := Parse(no_cmdline, env.fakeLookupenv)
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
	c, err := Parse(cmdline, no_env.fakeLookupenv)
	Convey("Value could not be set using command-line parameter", t, func() {
		So(c.WorkspaceDirectory, ShouldEqual, "/workspace")
		So(c.Verbose, ShouldBeTrue)
		So(err, ShouldBeNil)
	})
}
