// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package workspace

import (
	"context"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/detective"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/logeater/announcer"
	"gitlab.com/lightmeter/controlcenter/logeater/filelogsource"
	"gitlab.com/lightmeter/controlcenter/logeater/logsource"
	"gitlab.com/lightmeter/controlcenter/logeater/transform"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"os"
	"testing"
	"time"
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

			// needed to prevent the insights execution of blocking
			announcer.Skip(ws.ImportAnnouncer())

			cancel()

			So(done(), ShouldBeNil)

			So(ws.HasLogs(), ShouldBeFalse)
		})
	})
}

func TestDetective(t *testing.T) {
	Convey("Detective on real logs", t, func() {

		dir, clearDir := testutil.TempDir(t)
		defer clearDir()

		ws, err := NewWorkspace(dir)
		So(err, ShouldBeNil)

		defer ws.Close()

		year := 2020

		builder, err := transform.Get("default", year)
		So(err, ShouldBeNil)

		f, err := os.Open("../test_files/postfix_logs/individual_files/3_local_delivery.log")
		So(err, ShouldBeNil)

		logSource, err := filelogsource.New(f, builder, ws.ImportAnnouncer())
		So(err, ShouldBeNil)

		done, cancel := ws.Run()

		logReader := logsource.NewReader(logSource, ws.NewPublisher())

		err = logReader.Run()
		So(err, ShouldBeNil)

		cancel()
		err = done()
		So(err, ShouldBeNil)

		// actual Message Detective testing
		d := ws.Detective()

		Convey("Message found", func() {
			interval := timeutil.TimeInterval{
				time.Date(year, time.January, 0, 0, 0, 0, 0, time.Local),
				time.Date(year, time.December, 31, 0, 0, 0, 0, time.Local),
			}
			messages, err := d.CheckMessageDelivery(context.Background(), "sender@example.com", "recipient@example.com", interval)
			So(err, ShouldBeNil)

			expected_time := time.Date(year, time.January, 10, 16, 15, 30, 0, time.UTC)
			So(messages, ShouldResemble, []detective.MessageDelivery{detective.MessageDelivery{expected_time.In(time.UTC), "sent", "2.0.0"}})
		})

		Convey("Message out of interval", func() {
			interval := timeutil.TimeInterval{
				time.Date(year+1, time.January, 0, 0, 0, 0, 0, time.Local),
				time.Date(year+1, time.December, 31, 0, 0, 0, 0, time.Local),
			}
			messages, err := d.CheckMessageDelivery(context.Background(), "sender@example.com", "recipient@example.com", interval)
			So(err, ShouldBeNil)

			So(messages, ShouldResemble, []detective.MessageDelivery{})
		})
	})
}
