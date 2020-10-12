package meta

import (
	"errors"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"testing"
)

func init() {
	lmsqlite3.Initialize(lmsqlite3.Options{})
}

func TestRunner(t *testing.T) {
	Convey("Test Runner", t, func() {
		conn, closeConn := testutil.TempDBConnection()
		defer closeConn()

		handler, err := NewHandler(conn, "master")
		So(err, ShouldBeNil)

		defer func() {
			errorutil.MustSucceed(handler.Close())
		}()

		reader := handler.Reader

		runner := NewRunner(handler)

		done, cancel := runner.Run()

		Convey("Key not found", func() {
			_, err := reader.Retrieve(dummyContext, "key1")
			So(errors.Is(err, ErrNoSuchKey), ShouldBeTrue)
		})

		Convey("Insert multiple values", func() {
			err := runner.Store([]Item{
				{Key: "key1", Value: "value1"},
				{Key: "key2", Value: "value2"},
			}).Wait()

			So(err, ShouldBeNil)

			v, err := reader.Retrieve(dummyContext, "key1")
			So(err, ShouldBeNil)
			So(v, ShouldEqual, "value1")

			v, err = reader.Retrieve(dummyContext, "key2")
			So(err, ShouldBeNil)
			So(v, ShouldEqual, "value2")

			Convey("Update value", func() {
				err := runner.Store([]Item{{Key: "key1", Value: "another_value1"}}).Wait()
				So(err, ShouldBeNil)

				v, err := reader.Retrieve(dummyContext, "key1")
				So(err, ShouldBeNil)
				So(v, ShouldEqual, "another_value1")

				// key2 keeps the same value
				v, err = reader.Retrieve(dummyContext, "key2")
				So(err, ShouldBeNil)
				So(v, ShouldEqual, "value2")
			})

			cancel()
			done()
		})
	})
}
