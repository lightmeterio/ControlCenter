// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package metadata

import (
	"context"
	"errors"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/util/stringutil"
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
		conn, closeConn := testutil.TempDBConnectionMigrated(t, "master")
		defer closeConn()

		handler, err := NewHandler(conn)
		So(err, ShouldBeNil)

		reader := handler.Reader
		writer := handler.Writer

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
		conn, closeConn := testutil.TempDBConnectionMigrated(t, "master")
		defer closeConn()

		handler, err := NewHandler(conn)
		So(err, ShouldBeNil)

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

type testCategory int

const (
	category1 testCategory = 0
	category2 testCategory = 1
)

func (t *testCategory) MergoFromString(s string) error {
	switch s {
	case "cat1":
		*t = category1
		return nil
	case "cat2":
		*t = category2
		return nil
	}

	return errors.New(`Invalid test category`)
}

func TestOverridingValues(t *testing.T) {
	Convey("Test Overriding Values", t, func() {
		conn, closeConn := testutil.TempDBConnectionMigrated(t, "master")
		defer closeConn()

		defaultValues := DefaultValues{
			"key1": map[string]string{
				"subkey1": "alice",
				"subkey3": "bob",
				"subkey4": "clarice",
			},

			"key4": "some_default_string",
		}

		handler, err := NewDefaultedHandler(conn, defaultValues)
		So(err, ShouldBeNil)

		reader := handler.Reader
		writer := handler.Writer

		Convey("Json value not found", func() {
			var value []int
			err := reader.RetrieveJson(dummyContext, "key42", &value)
			So(errors.Is(err, ErrNoSuchKey), ShouldBeTrue)
		})

		Convey("Obtain simple value from defaults", func() {
			value, err := reader.Retrieve(dummyContext, "key4")
			So(err, ShouldBeNil)
			s, ok := value.(string)
			So(ok, ShouldBeTrue)
			So(s, ShouldEqual, "some_default_string")
		})

		Convey("Handle custom struct", func() {
			type myType struct {
				Category testCategory `json:"category"`
				Name     string       `json:"this_value_here_is_ignored"`
				Age      int          `json:"really_not_used_by_the_the_merging_library"`
			}

			handler, err := NewDefaultedHandler(conn, map[string]interface{}{
				"key": map[string]interface{}{
					"name":     "Some Name",
					"category": "cat2",
				},
			})

			So(err, ShouldBeNil)

			reader := handler.Reader
			writer := handler.Writer

			err = writer.StoreJson(dummyContext, "key", myType{Age: 42})
			So(err, ShouldBeNil)

			var value myType
			err = reader.RetrieveJson(dummyContext, "key", &value)
			So(err, ShouldBeNil)
			So(value, ShouldResemble, myType{Name: "Some Name", Age: 42, Category: category2})
		})

		Convey("Empty values do not override defaults", func() {
			type myType struct {
				Name    string               `json:"normal_name"`
				Surname stringutil.Sensitive `json:"secret_surname,omitempty"`
			}

			handler, err := NewDefaultedHandler(conn, map[string]interface{}{
				"key": map[string]interface{}{
					"name":    "Some Name",
					"surname": "Some Surname",
				},
			})

			So(err, ShouldBeNil)

			reader := handler.Reader
			writer := handler.Writer

			// the empty string should not override the default one!
			err = writer.StoreJson(dummyContext, "key", myType{Name: ""})
			So(err, ShouldBeNil)

			var value myType
			err = reader.RetrieveJson(dummyContext, "key", &value)
			So(err, ShouldBeNil)
			So(value, ShouldResemble, myType{Name: "Some Name", Surname: stringutil.MakeSensitive("Some Surname")})
		})

		Convey("Use value from defaults only", func() {
			var value map[string]string
			err := reader.RetrieveJson(dummyContext, "key1", &value)
			So(err, ShouldBeNil)
			So(value, ShouldResemble, map[string]string{
				"subkey1": "alice",
				"subkey3": "bob",
				"subkey4": "clarice",
			})
		})

		Convey("Use value from database only", func() {
			err := writer.StoreJson(dummyContext, "key2", map[string]int{
				"age": 24,
			})

			var value map[string]int
			err = reader.RetrieveJson(dummyContext, "key2", &value)
			So(err, ShouldBeNil)
			So(value, ShouldResemble, map[string]int{
				"age": 24,
			})
		})

		Convey("Override non Json value", func() {
			err := writer.Store(dummyContext, []Item{{Key: "key4", Value: "some_overriden_string"}})
			So(err, ShouldBeNil)

			value, err := reader.Retrieve(dummyContext, "key4")
			So(err, ShouldBeNil)
			s, ok := value.(string)
			So(ok, ShouldBeTrue)
			So(s, ShouldEqual, "some_overriden_string")
		})

		Convey("Change some of the values, adding others", func() {
			err := writer.StoreJson(dummyContext, "key1", map[string]string{
				"subkey1": "clara", // overriden
				"subkey2": "maria", // new value
				"subkey4": "",      // empty values do not override defaults
			})

			So(err, ShouldBeNil)

			var readValue map[string]string
			err = reader.RetrieveJson(dummyContext, "key1", &readValue)
			So(err, ShouldBeNil)
			So(readValue, ShouldResemble, map[string]string{
				"subkey1": "clara",
				"subkey2": "maria",
				"subkey3": "bob",
				"subkey4": "clarice",
			})
		})
	})
}
