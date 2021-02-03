// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0

package meta

import (
	"context"
	"errors"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
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
		conn, closeConn := testutil.TempDBConnection(t)
		defer closeConn()

		handler, err := NewHandler(conn, "master")
		So(err, ShouldBeNil)

		reader := handler.Reader
		writer := handler.Writer

		defer func() {
			errorutil.MustSucceed(handler.Close())
		}()

		Convey("Key not found", func() {
			_, err := reader.Retrieve(dummyContext, "key1")
			So(errors.Is(err, ErrNoSuchKey), ShouldBeTrue)
		})

		Convey("Insert multiple values", func() {
			err = writer.Store(dummyContext, []Item{
				{Key: "key1", Value: "value1"},
				{Key: "key2", Value: "value2"},
			})

			So(err, ShouldBeNil)

			v, err := reader.Retrieve(dummyContext, "key1")
			So(err, ShouldBeNil)
			So(v, ShouldEqual, "value1")

			v, err = reader.Retrieve(dummyContext, "key2")
			So(err, ShouldBeNil)
			So(v, ShouldEqual, "value2")

			Convey("Update value", func() {
				err := writer.Store(dummyContext, []Item{{Key: "key1", Value: "another_value1"}})
				So(err, ShouldBeNil)

				v, err := reader.Retrieve(dummyContext, "key1")
				So(err, ShouldBeNil)
				So(v, ShouldEqual, "another_value1")

				// key2 keeps the same value
				v, err = reader.Retrieve(dummyContext, "key2")
				So(err, ShouldBeNil)
				So(v, ShouldEqual, "value2")
			})
		})
	})
}

func TestJsonValues(t *testing.T) {
	Convey("Test Json Values", t, func() {
		conn, closeConn := testutil.TempDBConnection(t)
		defer closeConn()

		handler, err := NewHandler(conn, "master")
		So(err, ShouldBeNil)

		defer func() { errorutil.MustSucceed(handler.Close()) }()

		reader := handler.Reader
		writer := handler.Writer

		Convey("Key not found", func() {
			var value []int
			err := reader.RetrieveJson(dummyContext, "key1", &value)
			So(errors.Is(err, ErrNoSuchKey), ShouldBeTrue)
		})

		Convey("Insert array", func() {
			origValue := []int{1, 2, 3, 4}

			err := writer.StoreJson(dummyContext, "key1", origValue)
			So(err, ShouldBeNil)

			Convey("Successful read value", func() {
				var readValue []int
				err := reader.RetrieveJson(dummyContext, "key1", &readValue)
				So(err, ShouldBeNil)
				So(readValue, ShouldResemble, origValue)
			})

			Convey("Fail due wrong type", func() {
				var readValue []string
				err := reader.RetrieveJson(dummyContext, "key1", &readValue)
				So(err, ShouldNotBeNil)
			})

			Convey("Update value", func() {
				err := writer.StoreJson(dummyContext, "key1", []string{"one", "two"})
				So(err, ShouldBeNil)

				var retrieved []string
				err = reader.RetrieveJson(dummyContext, "key1", &retrieved)
				So(err, ShouldBeNil)
				So(retrieved, ShouldResemble, []string{"one", "two"})
			})
		})
	})
}
