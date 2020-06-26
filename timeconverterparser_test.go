package parser

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestTimeConverterParser(t *testing.T) {
	tz := time.UTC
	newYearNotifier := func(int, Time, Time) {}

	Convey("Test Contextual Parser", t, func() {
		Convey("Fails to parse", func() {
			Convey("Invalid Header", func() {
				c := NewTimeConverterParser(1999, tz, newYearNotifier)
				_, _, _, err := c.Parse([]byte(`invalid line`))
				So(err, ShouldEqual, ErrInvalidHeaderLine)
			})

			Convey("Invalid payload", func() {
				c := NewTimeConverterParser(1999, tz, newYearNotifier)
				t, _, _, err := c.Parse([]byte(`May 25 05:12:22 node process: Invalid Payload`))
				So(err, ShouldEqual, ErrUnsupportedLogLine)
				So(t.Unix(), ShouldEqual, 927609142)
			})
		})

		Convey("Year does not change", func() {
			c := NewTimeConverterParser(1999, tz, newYearNotifier)

			t, _, p, err := c.Parse([]byte(`May 25 05:12:22 smtpnode07 postfix-10.20.30.40/smtp[3022]: ` +
				`0C31D3D1E6: to=<redacted@aol.com>, relay=mx-aol.mail.gm0.yahoodns.net[11.22.33.44]:25, ` +
				`delay=18910, delays=18900/8.9/0.69/0.03, dsn=4.7.0, status=deferred extra message`))
			So(err, ShouldBeNil)
			So(p, ShouldNotBeNil)
			So(t.Unix(), ShouldEqual, 927609142)

			t, _, p, err = c.Parse([]byte(`May 25 05:12:24 smtpnode07 postfix-10.20.30.40/smtp[3022]: ` +
				`0C31D3D1E6: to=<redacted@aol.com>, relay=mx-aol.mail.gm0.yahoodns.net[11.22.33.44]:25, ` +
				`delay=18910, delays=18900/8.9/0.69/0.03, dsn=4.7.0, status=deferred extra message`))
			So(err, ShouldBeNil)
			So(p, ShouldNotBeNil)
			So(t.Unix(), ShouldEqual, 927609144)
		})

		Convey("Change year if the calendar changes", func() {
			c := NewTimeConverterParser(1999, tz, newYearNotifier)

			t, _, p, err := c.Parse([]byte(`Dec 31 23:59:58 smtpnode07 postfix-10.20.30.40/smtp[3022]: ` +
				`0C31D3D1E6: to=<redacted@aol.com>, relay=mx-aol.mail.gm0.yahoodns.net[11.22.33.44]:25, ` +
				`delay=18910, delays=18900/8.9/0.69/0.03, dsn=4.7.0, status=deferred extra message`))
			So(err, ShouldBeNil)
			So(p, ShouldNotBeNil)
			So(t.Unix(), ShouldEqual, 946684798)

			t, _, p, err = c.Parse([]byte(`Jan  1 00:00:00 smtpnode07 postfix-10.20.30.40/smtp[3022]: ` +
				`0C31D3D1E6: to=<redacted@aol.com>, relay=mx-aol.mail.gm0.yahoodns.net[11.22.33.44]:25, ` +
				`delay=18910, delays=18900/8.9/0.69/0.03, dsn=4.7.0, status=deferred extra message`))
			So(err, ShouldBeNil)
			So(p, ShouldNotBeNil)
			So(t.Unix(), ShouldEqual, 946684800)
		})
	})
}
