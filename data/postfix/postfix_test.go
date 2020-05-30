package postfix

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	parser "gitlab.com/lightmeter/postfix-log-parser"
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

	Convey("DST change", t, func() {
		t, err := time.Parse(`2006-01-02 15:04:05 -0700 MST`, `2020-03-01 06:44:32 +0100 CET`)

		if err != nil {
			panic("aaahhh")
		}

		t = t.In(time.UTC)

		initialTime := parser.Time{
			Month:  t.Month(),
			Day:    uint8(t.Day()),
			Hour:   uint8(t.Hour()),
			Minute: uint8(t.Minute()),
			Second: uint8(t.Second()),
		}

		c := NewTimeConverter(initialTime, t.Year(), t.Location(), func(int, parser.Time, parser.Time) {})
		c.Convert(parser.Time{Month: time.March, Day: 29, Hour: 2, Minute: 59, Second: 10})
		So(c.year, ShouldEqual, 2020)

		c.Convert(parser.Time{Month: time.March, Day: 29, Hour: 3, Minute: 5, Second: 17})
		So(c.year, ShouldEqual, 2020)
	})
}
