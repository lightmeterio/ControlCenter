package data

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestTimeInterval(t *testing.T) {
	Convey("Parse Time interval", t, func() {
		Convey("Fail to Parse interval begin", func() {
			_, err := ParseTimeInterval("lalala", "2010-01-01", time.UTC)
			So(err, ShouldNotEqual, nil)
		})

		Convey("Fail to parse interval end", func() {
			_, err := ParseTimeInterval("2000-01-01", "lalala", time.UTC)
			So(err, ShouldNotEqual, nil)
		})

		Convey("Parse Ordered Interval", func() {
			interval, err := ParseTimeInterval("2020-03-23", "2020-05-17", time.UTC)
			So(err, ShouldEqual, nil)
			So(interval.From.Unix(), ShouldEqual, 1584921600)
			So(interval.To.Unix(), ShouldEqual, 1589760000-1) // next day at midnight - 1
		})

		Convey("Fail to parse out of order Interval", func() {
			_, err := ParseTimeInterval("2020-05-17", "2020-03-23", time.UTC)
			So(err, ShouldEqual, ErrOutOfOrderTimeInterval)
		})
	})
}
