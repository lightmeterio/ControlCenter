// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package config

import (
	"os"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

var no_cmdline = []string{}

func TestDefaultValueWorkspace(t *testing.T) {
	c, err := ParseConfig(no_cmdline, os.LookupEnv)
	Convey("Incorrect default value for parameter 'workspace'", t, func() {
		So(c.WorkspaceDirectory, ShouldEqual, "/var/lib/lightmeter_workspace")
		So(err, ShouldBeNil)
	})
}

func TestEnvWorskpace(t *testing.T) {
	os.Setenv("LIGHTMETER_WORKSPACE", "/sdjklhfjksd")
	c, err := ParseConfig(no_cmdline, os.LookupEnv)
	Convey("Value could not be set using string environment variable 'LIGHTMETER_WORKSPACE'", t, func() {
		So(c.WorkspaceDirectory, ShouldEqual, "/sdjklhfjksd")
		So(err, ShouldBeNil)
	})
}

func TestEnvVerbose(t *testing.T) {
	os.Setenv("LIGHTMETER_VERBOSE", "True")
	c, err := ParseConfig(no_cmdline, os.LookupEnv)
	Convey("Value could not be set using boolean environment variable 'LIGHTMETER_VERBOSE'", t, func() {
		So(c.Verbose, ShouldBeTrue)
		So(err, ShouldBeNil)
	})
}
