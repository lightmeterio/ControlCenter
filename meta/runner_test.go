// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package meta

import (
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"testing"
)

func init() {
	lmsqlite3.Initialize(lmsqlite3.Options{})
}

func TestRunner(t *testing.T) {
	Convey("Test Runner", t, func() {
		_, closeDatabases := testutil.TempDatabases(t)
		defer closeDatabases()

		db := dbconn.Db("master")

		runner := NewRunner(db)

		done, cancel := runner.Run()

		defer func() { cancel(); done() }()

		writer := runner.Writer()

		Convey("Insert multiple values", func() {
			err := writer.Store([]Item{
				{Key: "key1", Value: "value1"},
				{Key: "key2", Value: "value2"},
			}).Wait()

			So(err, ShouldBeNil)

			var v string
			err = Retrieve(dummyContext, db, "key1", &v)
			So(err, ShouldBeNil)
			So(v, ShouldEqual, "value1")

			err = Retrieve(dummyContext, db, "key2", &v)
			So(err, ShouldBeNil)
			So(v, ShouldEqual, "value2")

			Convey("Update value", func() {
				err := writer.Store([]Item{{Key: "key1", Value: "another_value1"}}).Wait()
				So(err, ShouldBeNil)

				err = Retrieve(dummyContext, db, "key1", &v)
				So(err, ShouldBeNil)
				So(v, ShouldEqual, "another_value1")

				// key2 keeps the same value
				err = Retrieve(dummyContext, db, "key2", &v)
				So(err, ShouldBeNil)
				So(v, ShouldEqual, "value2")
			})

			Convey("Insert array", func() {
				origValue := []int{1, 2, 3, 4}

				err := writer.StoreJson("key1", origValue).Wait()
				So(err, ShouldBeNil)

				Convey("Successful read value", func() {
					var readValue []int
					err := RetrieveJson(dummyContext, db, "key1", &readValue)
					So(err, ShouldBeNil)
					So(readValue, ShouldResemble, origValue)
				})

				Convey("Update value", func() {
					err := writer.StoreJson("key1", []string{"one", "two"}).Wait()
					So(err, ShouldBeNil)

					var retrieved []string
					err = RetrieveJson(dummyContext, db, "key1", &retrieved)
					So(err, ShouldBeNil)
					So(retrieved, ShouldResemble, []string{"one", "two"})
				})
			})
		})
	})
}
