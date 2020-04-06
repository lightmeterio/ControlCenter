package workspace

import (
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/data"
	"io/ioutil"
	"os"
	"testing"
	"time"
)

func tempDir() string {
	dir, e := ioutil.TempDir("", "lightmeter-tests-*")
	if e != nil {
		panic("error creating temp dir")
	}
	return dir
}

func TestWorkspaceCreation(t *testing.T) {
	Convey("Creation fails on several scenarios", t, func() {
		Convey("No Permission on workspace", func() {
			// FIXME: this is relying on linux properties, as /proc is a read-only directory
			_, err := NewWorkspace("/proc/lalala", data.Config{Location: time.UTC, DefaultYear: 1999})
			So(err, ShouldNotEqual, nil)
		})
	})

	Convey("Creation succeeds", t, func() {
		Convey("Create Workspace", func() {
			dir := tempDir()
			defer os.RemoveAll(dir)
			ws, err := NewWorkspace(dir, data.Config{Location: time.UTC, DefaultYear: 1999})
			So(err, ShouldEqual, nil)

			defer ws.Close()
			So(ws.HasLogs(), ShouldBeFalse)
		})

		Convey("Empty Database is properly closed", func() {
			dir := tempDir()
			defer os.RemoveAll(dir)
			ws, err := NewWorkspace(dir, data.Config{Location: time.UTC, DefaultYear: 1999})
			So(err, ShouldEqual, nil)
			So(ws.HasLogs(), ShouldBeFalse)
			So(ws.Close(), ShouldEqual, nil)
		})

		Convey("Reopening workspace succeeds", func() {
			dir := tempDir()
			defer os.RemoveAll(dir)

			ws1, err := NewWorkspace(dir, data.Config{Location: time.UTC, DefaultYear: 1999})
			ws1.Close()

			ws2, err := NewWorkspace(dir, data.Config{Location: time.UTC, DefaultYear: 1999})
			So(err, ShouldEqual, nil)
			ws2.Close()
		})
	})
}
