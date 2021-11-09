// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package timeutil

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestTimeInterval(t *testing.T) {
	Convey("Parse Time interval", t, func() {
		Convey("Fail to Parse interval begin", func() {
			_, err := ParseTimeInterval("lalala", "2010-01-01", time.UTC)
			So(err, ShouldNotBeNil)
		})

		Convey("Invalid interval component", func() {
			_, err := ParseTimeInterval("2000-01-01", "lalala", time.UTC)
			So(err, ShouldNotBeNil)

			_, err = ParseTimeInterval("2000-01-0a", "2000-01-04", time.UTC)
			So(err, ShouldNotBeNil)

			_, err = ParseTimeInterval("2000-01-01", "2000-01-04 AA:00:3K", time.UTC)
			So(err, ShouldNotBeNil)
		})

		Convey("Parse Ordered Interval", func() {
			interval, err := ParseTimeInterval("2020-03-23", "2020-05-17", time.UTC)
			So(err, ShouldBeNil)
			So(interval.From, ShouldResemble, MustParseTime(`2020-03-23 00:00:00 +0000`))
			So(interval.To, ShouldResemble, MustParseTime(`2020-05-17 23:59:59 +0000`))
		})

		Convey("Parse Ordered Interval with time component", func() {
			interval, err := ParseTimeInterval("2020-03-23 10:34:45", "2020-05-17", time.UTC)
			So(err, ShouldBeNil)
			So(interval.From, ShouldResemble, MustParseTime(`2020-03-23 10:34:45 +0000`))
			So(interval.To, ShouldResemble, MustParseTime(`2020-05-17 23:59:59 +0000`))
		})

		Convey("Fail to parse out of order Interval", func() {
			_, err := ParseTimeInterval("2020-05-17", "2020-03-23", time.UTC)
			So(err, ShouldEqual, ErrOutOfOrderTimeInterval)
		})
	})
}
