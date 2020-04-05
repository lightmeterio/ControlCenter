package postfix

import (
	. "github.com/smartystreets/goconvey/convey"
	parser "gitlab.com/lightmeter/postfix-log-parser"
	"testing"
	"time"
)

func TestPostfixTimeConverter(t *testing.T) {
	tz := time.UTC

	newYearNotifier := func(int, parser.Time, parser.Time) {}

	Convey("With zeroed initial time", t, func() {
		initialTime := parser.Time{}

		Convey("Calls without changing year", func() {
			c := NewTimeConverter(initialTime, 1999, tz, newYearNotifier)
			So(c.Convert(parser.Time{Month: time.May, Day: 25, Hour: 5, Minute: 12, Second: 22}).Unix(), ShouldEqual, 927609142)
			So(c.year, ShouldEqual, 1999)
			So(c.Convert(parser.Time{Month: time.May, Day: 25, Hour: 5, Minute: 12, Second: 22}).Unix(), ShouldEqual, 927609142)
			So(c.year, ShouldEqual, 1999)
			So(c.Convert(parser.Time{Month: time.December, Day: 31, Hour: 23, Minute: 59, Second: 59}).Unix(), ShouldEqual, 946684799)
			So(c.year, ShouldEqual, 1999)
		})

		Convey("Change year if the calendar changes", func() {
			c := NewTimeConverter(initialTime, 1999, tz, newYearNotifier)
			So(c.Convert(parser.Time{Month: time.December, Day: 31, Hour: 23, Minute: 59, Second: 58}).Unix(), ShouldEqual, 946684798)
			So(c.year, ShouldEqual, 1999)
			So(c.Convert(parser.Time{Month: time.January, Day: 1, Hour: 0, Minute: 0, Second: 0}).Unix(), ShouldEqual, 946684800)
			So(c.year, ShouldEqual, 2000)
		})
	})

	Convey("With non zero initial time", t, func() {
		initialTime := parser.Time{Month: time.February, Day: 21, Hour: 14, Minute: 52, Second: 34}

		Convey("Calls without changing year", func() {
			c := NewTimeConverter(initialTime, 1999, tz, newYearNotifier)
			So(c.Convert(parser.Time{Month: time.May, Day: 25, Hour: 5, Minute: 12, Second: 22}).Unix(), ShouldEqual, 927609142)
			So(c.year, ShouldEqual, 1999)
		})

		Convey("Calls changing year", func() {
			c := NewTimeConverter(initialTime, 1999, tz, newYearNotifier)
			So(c.Convert(parser.Time{Month: time.January, Day: 1, Hour: 0, Minute: 0, Second: 0}).Unix(), ShouldEqual, 946684800)
			So(c.year, ShouldEqual, 2000)
		})
	})
}
