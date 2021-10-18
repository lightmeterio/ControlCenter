// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package transform

import (
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"testing"
	"time"
)

func TestTransformers(t *testing.T) {
	Convey("Test Transformers", t, func() {
		Convey("Unknown", func() {
			_, err := Get("invalid blabla")
			So(err, ShouldEqual, ErrUnknownTransformer)
		})

		Convey("Invalid year on default transformer", func() {
			_, err := Get("default", "non-integer-value")
			So(err, ShouldNotBeNil)
		})

		Convey("Default no year passed, use current one", func() {
			builder, err := Get("default")
			So(err, ShouldBeNil)
			transformer, err := builder()
			So(err, ShouldBeNil)
			r, err := transformer.Transform(string(`Aug 21 03:03:04 mail dog: Useless Payload`))
			So(err, ShouldBeNil)
			So(r.Header.Host, ShouldEqual, "mail")
			So(r.Time, ShouldResemble, time.Date(time.Now().Year(), time.August, 21, 3, 3, 4, 0, time.UTC))
			So(r.Line, ShouldEqual, `Aug 21 03:03:04 mail dog: Useless Payload`)
		})

		Convey("Default, just return the line, unable to get a time from it", func() {
			builder, err := Get("default", 2000)
			So(err, ShouldBeNil)
			transformer, err := builder()
			So(err, ShouldBeNil)
			r, err := transformer.Transform(string(`Aug 21 03:03:04 mail dog: Useless Payload`))
			So(err, ShouldBeNil)
			So(r.Header.Host, ShouldEqual, "mail")
			So(r.Time, ShouldResemble, testutil.MustParseTime(`2000-08-21 03:03:04 +0000`))
		})

		Convey("Default, just return the line, use current year as the passed one is zero", func() {
			builder, err := Get("default", 0)
			So(err, ShouldBeNil)
			transformer, err := builder()
			So(err, ShouldBeNil)
			r, err := transformer.Transform(string(`Aug 21 03:03:04 mail dog: Useless Payload`))
			So(err, ShouldBeNil)
			So(r.Header.Host, ShouldEqual, "mail")
			So(r.Time, ShouldResemble, time.Date(time.Now().Year(), time.August, 21, 3, 3, 4, 0, time.UTC))
		})
	})
}

func TestRFC3339PrependFormat(t *testing.T) {
	Convey("Prepent RFC3339", t, func() {
		builder, err := Get("prepend-rfc3339")
		So(err, ShouldBeNil)

		transformer, err := builder()
		So(err, ShouldBeNil)

		Convey("Fails invalid time and line", func() {
			_, err := transformer.Transform(string(`9898789`))
			So(err, ShouldNotBeNil)
		})

		Convey("Fails invalid time format", func() {
			_, err := transformer.Transform(string(`lalala Mar  6 07:08:59 host postfix/qmgr[28829]: A1E1E1880093: removed`))
			So(err, ShouldNotBeNil)
		})

		Convey("Fails invalid line", func() {
			_, err := transformer.Transform(string(`2021-03-06T06:09:00.798Z invalid line`))
			So(err, ShouldNotBeNil)
		})

		Convey("Fails ending with space", func() {
			_, err := transformer.Transform(string(`2021-03-06T06:09:00.798Z`))
			So(err, ShouldNotBeNil)
		})

		Convey("Succeeds", func() {
			r, err := transformer.Transform(string(`2021-03-06T06:09:00.000Z Mar  6 07:08:59 host postfix/qmgr[28829]: A1E1E1880093: removed`))
			So(err, ShouldBeNil)
			So(r.Time, ShouldResemble, testutil.MustParseTime(`2021-03-06 06:09:00 +0000`))
			So(r.Header.Host, ShouldEqual, "host")
			So(r.Line, ShouldEqual, `Mar  6 07:08:59 host postfix/qmgr[28829]: A1E1E1880093: removed`)
		})

		Convey("Succeeds, docker logs default format", func() {
			r, err := transformer.Transform(string(`2021-03-08T07:21:23.496826493Z Mar  8 07:21:23 mail postfix/anvil[2990]: statistics: max cache size 2 at Mar  8 07:18:01`))
			So(err, ShouldBeNil)
			So(r.Time, ShouldResemble, time.Date(2021, time.March, 8, 7, 21, 23, 496826493, time.UTC))
			So(r.Header.Host, ShouldEqual, "mail")
		})

	})
}

func TestLogstashJSON(t *testing.T) {
	Convey("Test logstash json logs", t, func() {
		builder, err := Get("logstash")
		So(err, ShouldBeNil)

		transformer, err := builder()
		So(err, ShouldBeNil)

		Convey("Invalid json payload", func() {
			_, err := transformer.Transform(string(`{{---`))
			So(err, ShouldNotBeNil)
		})

		Convey("Succeeds", func() {
			r, err := transformer.Transform(string(`{"log-source":"filebeat","@version":"1","input":{"type":"log"},"ecs":{"version":"1.6.0"},"message":"Mar 20 07:54:52 mail postfix/smtp[6807]: 586711880093: to=<XXXXXXXX>, relay=XXXXX[XXXXX]:25, delay=4.1, delays=0.15/0.01/1.4/2.5, dsn=2.0.0, status=sent (250 2.0.0 Ok: queued as 6ECB0A8019A)","log-type":"mail","tags":["beats_input_codec_plain_applied"],"type":"debug","hostname":"melian","@timestamp":"2021-03-20T06:54:55.835Z","log":{"file":{"path":"/var/log/mail.log"},"offset":4020961}}`))
			So(err, ShouldBeNil)
			expectedTime, err := time.Parse(time.RFC3339, `2021-03-20T06:54:55.835Z`)
			So(err, ShouldBeNil)
			So(r.Time, ShouldResemble, expectedTime)
			So(r.Header.Host, ShouldEqual, "mail")
			So(r.Location.Filename, ShouldEqual, "/var/log/mail.log")
			So(r.Line, ShouldEqual, `Mar 20 07:54:52 mail postfix/smtp[6807]: 586711880093: to=<XXXXXXXX>, relay=XXXXX[XXXXX]:25, delay=4.1, delays=0.15/0.01/1.4/2.5, dsn=2.0.0, status=sent (250 2.0.0 Ok: queued as 6ECB0A8019A)`)
		})
	})
}

func TestRFC3339(t *testing.T) {
	Convey("Test RFC3339", t, func() {
		builder, err := Get("rfc3339")
		So(err, ShouldBeNil)

		transformer, err := builder()
		So(err, ShouldBeNil)

		Convey("Succeeds", func() {
			r, err := transformer.Transform(string(`2021-05-16T00:01:44.278515+02:00 mail postfix/postscreen[17274]: Useless Payload`))
			So(err, ShouldBeNil)
			expectedTime := timeutil.MustParseTime(`2021-05-16 00:01:44 +0000`)
			So(r.Time, ShouldResemble, expectedTime)
			So(r.Header.Host, ShouldEqual, "mail")
		})
	})
}
