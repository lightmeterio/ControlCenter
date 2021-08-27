// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package meta

import (
	"context"
	"errors"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"testing"
)

var (
	dummyContext = context.Background()
)

func init() {
	lmsqlite3.Initialize(lmsqlite3.Options{})
}

func TestSimpleValues(t *testing.T) {
	Convey("Test Meta", t, func() {
		_, closeDatabases := testutil.TempDatabases(t)
		defer closeDatabases()

		db := dbconn.Db("master")

		Convey("Key not found", func() {
			var v interface{}
			err := Retrieve(dummyContext, db, "key1", &v)
			So(errors.Is(err, ErrNoSuchKey), ShouldBeTrue)
		})

		Convey("Insert multiple values", func() {
			err := Store(dummyContext, db, []Item{
				{Key: "key1", Value: "value1"},
				{Key: "key2", Value: "value2"},
			})

			So(err, ShouldBeNil)

			var v string
			err = Retrieve(dummyContext, db, "key1", &v)
			So(err, ShouldBeNil)
			So(v, ShouldEqual, "value1")

			err = Retrieve(dummyContext, db, "key2", &v)
			So(err, ShouldBeNil)
			So(v, ShouldEqual, "value2")

			Convey("Update value", func() {
				err := Store(dummyContext, db, []Item{{Key: "key1", Value: "another_value1"}})
				So(err, ShouldBeNil)

				err = Retrieve(dummyContext, db, "key1", &v)
				So(err, ShouldBeNil)
				So(v, ShouldEqual, "another_value1")

				// key2 keeps the same value
				err = Retrieve(dummyContext, db, "key2", &v)
				So(err, ShouldBeNil)
				So(v, ShouldEqual, "value2")
			})
		})
	})
}

func TestJsonValues(t *testing.T) {
	Convey("Test Json Values", t, func() {
		_, closeDatabases := testutil.TempDatabases(t)
		defer closeDatabases()

		db := dbconn.Db("master")

		Convey("Key not found", func() {
			var value []int
			err := RetrieveJson(dummyContext, db, "key1", &value)
			So(errors.Is(err, ErrNoSuchKey), ShouldBeTrue)
		})

		Convey("Insert array", func() {
			origValue := []int{1, 2, 3, 4}

			err := StoreJson(dummyContext, db, "key1", origValue)
			So(err, ShouldBeNil)

			Convey("Successful read value", func() {
				var readValue []int
				err := RetrieveJson(dummyContext, db, "key1", &readValue)
				So(err, ShouldBeNil)
				So(readValue, ShouldResemble, origValue)
			})

			Convey("Fail due wrong type", func() {
				var readValue []string
				err := RetrieveJson(dummyContext, db, "key1", &readValue)
				So(err, ShouldNotBeNil)
			})

			Convey("Update value", func() {
				err := StoreJson(dummyContext, db, "key1", []string{"one", "two"})
				So(err, ShouldBeNil)

				var retrieved []string
				err = RetrieveJson(dummyContext, db, "key1", &retrieved)
				So(err, ShouldBeNil)
				So(retrieved, ShouldResemble, []string{"one", "two"})
			})
		})
	})
}
