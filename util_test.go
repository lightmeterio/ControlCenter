package parser

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestPostfixTimeConverter(t *testing.T) {
	tz := time.UTC

	newYearNotifier := func(int, Time, Time) {}

	Convey("With zeroed initial time", t, func() {
		initialTime := DefaultTimeInYear(1999, tz)

		Convey("Calls without changing year", func() {
			c := NewTimeConverter(initialTime, newYearNotifier)
			So(c.Convert(Time{Month: time.May, Day: 25, Hour: 5, Minute: 12, Second: 22}).Unix(), ShouldEqual, 927609142)
			So(c.year, ShouldEqual, 1999)
			So(c.Convert(Time{Month: time.May, Day: 25, Hour: 5, Minute: 12, Second: 22}).Unix(), ShouldEqual, 927609142)
			So(c.year, ShouldEqual, 1999)
			So(c.Convert(Time{Month: time.December, Day: 31, Hour: 23, Minute: 59, Second: 59}).Unix(), ShouldEqual, 946684799)
			So(c.year, ShouldEqual, 1999)
		})

		Convey("Change year if the calendar changes", func() {
			c := NewTimeConverter(initialTime, newYearNotifier)
			So(c.Convert(Time{Month: time.December, Day: 31, Hour: 23, Minute: 59, Second: 58}).Unix(), ShouldEqual, 946684798)
			So(c.year, ShouldEqual, 1999)
			So(c.Convert(Time{Month: time.January, Day: 1, Hour: 0, Minute: 0, Second: 0}).Unix(), ShouldEqual, 946684800)
			So(c.year, ShouldEqual, 2000)
		})
	})

	Convey("With non zero initial time", t, func() {
		initialTime := time.Date(1999, time.February, 20, 14, 52, 34, 0, tz)

		Convey("Calls without changing year", func() {
			c := NewTimeConverter(initialTime, newYearNotifier)
			So(c.Convert(Time{Month: time.May, Day: 25, Hour: 5, Minute: 12, Second: 22}).Unix(), ShouldEqual, 927609142)
			So(c.year, ShouldEqual, 1999)
		})

		Convey("Calls changing year", func() {
			c := NewTimeConverter(initialTime, newYearNotifier)
			So(c.Convert(Time{Month: time.January, Day: 1, Hour: 0, Minute: 0, Second: 0}).Unix(), ShouldEqual, 946684800)
			So(c.year, ShouldEqual, 2000)
		})
	})

	Convey("DST change", t, func() {
		t, err := time.Parse(`2006-01-02 15:04:05 -0700 MST`, `2020-03-01 06:44:32 +0100 CET`)

		if err != nil {
			panic("aaahhh")
		}

		t = t.In(time.UTC)

		c := NewTimeConverter(t, func(int, Time, Time) {})
		c.Convert(Time{Month: time.March, Day: 29, Hour: 2, Minute: 59, Second: 10})
		So(c.year, ShouldEqual, 2020)

		c.Convert(Time{Month: time.March, Day: 29, Hour: 3, Minute: 5, Second: 17})
		So(c.year, ShouldEqual, 2020)
	})
}
