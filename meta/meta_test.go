package meta

import (
	"errors"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"path"
	"testing"
)

func init() {
	lmsqlite3.Initialize(lmsqlite3.Options{})
}

func TestSimpleValues(t *testing.T) {
	Convey("Test Meta", t, func() {
		dir, clearDir := testutil.TempDir()
		defer clearDir()

		conn, err := dbconn.NewConnPair(path.Join(dir, "master.db"))

		handler, err := NewMetaDataHandler(conn, "master")
		So(err, ShouldBeNil)
		defer func() { errorutil.MustSucceed(handler.Close()) }()

		Convey("Key not found", func() {
			_, err := handler.Retrieve("key1")
			So(errors.Is(err, ErrNoSuchKey), ShouldBeTrue)
		})

		Convey("Insert multiple values", func() {
			_, err = handler.Store([]Item{
				{Key: "key1", Value: "value1"},
				{Key: "key2", Value: "value2"},
			})

			So(err, ShouldBeNil)

			v, err := handler.Retrieve("key1")
			So(err, ShouldBeNil)
			So(v, ShouldEqual, "value1")

			v, err = handler.Retrieve("key2")
			So(err, ShouldBeNil)
			So(v, ShouldEqual, "value2")

			Convey("Update value", func() {
				_, err := handler.Store([]Item{{Key: "key1", Value: "another_value1"}})
				So(err, ShouldBeNil)

				v, err := handler.Retrieve("key1")
				So(err, ShouldBeNil)
				So(v, ShouldEqual, "another_value1")

				// key2 keeps the same value
				v, err = handler.Retrieve("key2")
				So(err, ShouldBeNil)
				So(v, ShouldEqual, "value2")
			})
		})
	})
}

func TestJsonValues(t *testing.T) {
	Convey("Test Json Values", t, func() {
		dir, clearDir := testutil.TempDir()
		defer clearDir()

		conn, err := dbconn.NewConnPair(path.Join(dir, "master.db"))

		handler, err := NewMetaDataHandler(conn, "master")
		So(err, ShouldBeNil)
		defer func() { errorutil.MustSucceed(handler.Close()) }()

		Convey("Key not found", func() {
			var value []int
			err := handler.RetrieveJson("key1", &value)
			So(errors.Is(err, ErrNoSuchKey), ShouldBeTrue)
		})

		Convey("Insert array", func() {
			origValue := []int{1, 2, 3, 4}

			_, err = handler.StoreJson("key1", origValue)
			So(err, ShouldBeNil)

			Convey("Successful read value", func() {
				var readValue []int
				err := handler.RetrieveJson("key1", &readValue)
				So(err, ShouldBeNil)
				So(readValue, ShouldResemble, origValue)
			})

			Convey("Fail due wrong type", func() {
				var readValue []string
				err := handler.RetrieveJson("key1", &readValue)
				So(err, ShouldNotBeNil)
			})

			Convey("Update value", func() {
				_, err = handler.StoreJson("key1", []string{"one", "two"})
				So(err, ShouldBeNil)

				var retrieved []string
				err := handler.RetrieveJson("key1", &retrieved)
				So(err, ShouldBeNil)
				So(retrieved, ShouldResemble, []string{"one", "two"})
			})
		})
	})
}
