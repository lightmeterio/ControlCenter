// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0

package workspace

import (
	"errors"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/util/closeutil"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"io"
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

		Convey("Close workspace failed", func() {
			dir, clearDir := testutil.TempDir(t)
			defer clearDir()

			ws1, err := NewWorkspace(dir)
			So(err, ShouldBeNil)

			ws1.closes = []io.Closer{
				closeutil.ConvertToCloser(func() error {
					return errorutil.Wrap(errors.New("closes 1"))
				}),
				closeutil.ConvertToCloser(func() error {
					return errorutil.Wrap(errors.New("closes 2"))
				}),
				closeutil.ConvertToCloser(func() error {
					return errorutil.Wrap(errors.New("closes 3"))
				}),
			}

			err = ws1.Close()
			So(err, ShouldNotBeNil)
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

			cancel()

			So(done(), ShouldBeNil)

			So(ws.HasLogs(), ShouldBeFalse)
		})
	})
}
