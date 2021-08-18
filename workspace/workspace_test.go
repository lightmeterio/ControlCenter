// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package workspace

import (
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/logeater/announcer"
	"gitlab.com/lightmeter/controlcenter/util/postfixutil"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"testing"
)

func init() {
	lmsqlite3.Initialize(lmsqlite3.Options{})
}

func TestWorkspaceCreation(t *testing.T) {
	Convey("Creation fails on several scenarios", t, func() {
		Convey("No Permission on workspace", func() {
			// FIXME: this is relying on linux properties, as /proc is a read-only directory
			_, err := NewWorkspace("/proc/lalala")
			So(err, ShouldNotBeNil)
		})
	})

	Convey("Creation succeeds", t, func() {
		Convey("Create Workspace", func() {
			dir, clearDir := testutil.TempDir(t)
			defer clearDir()

			ws, err := NewWorkspace(dir)
			So(err, ShouldBeNil)

			defer ws.Close()
			So(ws.HasLogs(), ShouldBeFalse)
		})

		Convey("Empty Database is properly closed", func() {
			dir, clearDir := testutil.TempDir(t)
			defer clearDir()

			ws, err := NewWorkspace(dir)
			So(err, ShouldBeNil)
			So(ws.HasLogs(), ShouldBeFalse)
			So(ws.Close(), ShouldBeNil)
		})

		Convey("Reopening workspace succeeds", func() {
			dir, clearDir := testutil.TempDir(t)
			defer clearDir()

			ws1, err := NewWorkspace(dir)
			So(err, ShouldBeNil)
			ws1.Close()

			ws2, err := NewWorkspace(dir)
			So(err, ShouldBeNil)
			ws2.Close()
		})
	})
}

func TestWorkspaceExecution(t *testing.T) {
	Convey("Workspace execution", t, func() {
		Convey("Nothing read", func() {
			dir, clearDir := testutil.TempDir(t)
			defer clearDir()

			ws, err := NewWorkspace(dir)
			So(err, ShouldBeNil)

			defer ws.Close()

			done, cancel := ws.Run()

			importAnnouncer, err := ws.ImportAnnouncer()
			So(err, ShouldBeNil)

			// needed to prevent the insights execution of blocking
			announcer.Skip(importAnnouncer)

			cancel()

			So(done(), ShouldBeNil)

			So(ws.HasLogs(), ShouldBeFalse)
		})
	})
}

func TestMostRecentLogTime(t *testing.T) {
	Convey("MostRecentLogTime", t, func() {
		dir, clearDir := testutil.TempDir(t)
		defer clearDir()

		ws, err := NewWorkspace(dir)
		So(err, ShouldBeNil)

		defer ws.Close()

		done, cancel := ws.Run()

		// needed to prevent the insights execution of blocking
		announcer.Skip(ws.ImportAnnouncer())

		postfixutil.ReadFromTestFile("../test_files/postfix_logs/individual_files/1_bounce_simple.log", ws.NewPublisher(), 2020)

		cancel()

		So(done(), ShouldBeNil)

		mostRecentTime, err := ws.MostRecentLogTime()
		So(err, ShouldBeNil)
		So(mostRecentTime, ShouldResemble, timeutil.MustParseTime(`2020-06-03 10:40:59 +0000`))

		So(ws.HasLogs(), ShouldBeTrue)
	})
}
