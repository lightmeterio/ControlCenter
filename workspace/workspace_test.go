package workspace

import (
	"errors"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/logdb"
	"gitlab.com/lightmeter/controlcenter/util/closeutil"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"io"
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
			_, err := NewWorkspace("/proc/lalala", logdb.Config{Location: time.UTC})
			So(err, ShouldNotBeNil)
		})
	})

	Convey("Creation succeeds", t, func() {
		Convey("Create Workspace", func() {

			dir, clearDir := testutil.TempDir()
			defer clearDir()

			ws, err := NewWorkspace(dir, logdb.Config{Location: time.UTC})
			So(err, ShouldBeNil)

			defer ws.Close()
			So(ws.HasLogs(), ShouldBeFalse)
		})

		Convey("Empty Database is properly closed", func() {
			dir, clearDir := testutil.TempDir()
			defer clearDir()

			ws, err := NewWorkspace(dir, logdb.Config{Location: time.UTC})
			So(err, ShouldBeNil)
			So(ws.HasLogs(), ShouldBeFalse)
			So(ws.Close(), ShouldBeNil)
		})

		Convey("Reopening workspace succeeds", func() {
			dir, clearDir := testutil.TempDir()
			defer clearDir()

			ws1, err := NewWorkspace(dir, logdb.Config{Location: time.UTC})
			ws1.Close()

			ws2, err := NewWorkspace(dir, logdb.Config{Location: time.UTC})
			So(err, ShouldBeNil)
			ws2.Close()
		})

		Convey("Close workspace failed", func() {
			dir, clearDir := testutil.TempDir()
			defer clearDir()

			ws1, err := NewWorkspace(dir, logdb.Config{Location: time.UTC})
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
