// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package workspace

import (
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/logeater/announcer"
	"gitlab.com/lightmeter/controlcenter/pkg/runner"
	"gitlab.com/lightmeter/controlcenter/util/postfixutil"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"strings"
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

			// needed to prevent the insights execution of blocking
			importAnnouncer, err := ws.ImportAnnouncer()
			So(err, ShouldBeNil)
			announcer.Skip(importAnnouncer)

			done, cancel := runner.Run(ws)

			cancel()

			So(done(), ShouldBeNil)

			So(ws.HasLogs(), ShouldBeFalse)
		})

		So(dbconn.CountOpenConnections(), ShouldEqual, 0)
		So(dbconn.CountDetails(), ShouldResemble, map[dbconn.CounterKey]int{})
	})
}

func TestMostRecentLogTime(t *testing.T) {
	Convey("MostRecentLogTime", t, func() {
		dir, clearDir := testutil.TempDir(t)
		defer clearDir()

		Convey("Basic case", func() {
			ws, err := NewWorkspace(dir)
			So(err, ShouldBeNil)

			defer ws.Close()

			// needed to prevent the insights execution of blocking
			importAnnouncer, err := ws.ImportAnnouncer()
			So(err, ShouldBeNil)
			announcer.Skip(importAnnouncer)

			done, cancel := runner.Run(ws)

			pub := ws.NewPublisher()

			postfixutil.ReadFromTestFile("../test_files/postfix_logs/individual_files/1_bounce_simple.log", pub, 2020)

			// then read some random info for failed connections (gitlab issue #548)
			postfixutil.ReadFromTestReader(strings.NewReader(
				`
Jun  3 10:41:05 mail postfix/smtpd[11978]: disconnect from unknown[1.2.3.4] ehlo=1 auth=0/1 commands=1/2
Jun  3 10:41:10 mail postfix/smtpd[11978]: disconnect from unknown[4.3.2.1] ehlo=1 auth=0/3 commands=1/3
`), pub, 2020)

			cancel()

			So(done(), ShouldBeNil)

			mostRecentTime, err := ws.MostRecentLogTime()
			So(err, ShouldBeNil)
			So(mostRecentTime, ShouldResemble, timeutil.MustParseTime(`2020-06-03 10:41:10 +0000`))

			So(ws.HasLogs(), ShouldBeTrue)
		})
	})
}
