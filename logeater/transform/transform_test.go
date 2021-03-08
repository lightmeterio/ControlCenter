// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package transform

import (
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
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
			r, err := transformer.Transform([]byte(`Aug 21 03:03:04 mail dog: Useless Payload`))
			So(err, ShouldBeNil)
			So(r.Header.Host, ShouldEqual, "mail")
			So(r.Time, ShouldResemble, time.Date(time.Now().Year(), time.August, 21, 3, 3, 4, 0, time.UTC))
		})

		Convey("Default, just return the line, unable to get a time from it", func() {
			builder, err := Get("default", 2000)
			So(err, ShouldBeNil)
			transformer, err := builder()
			So(err, ShouldBeNil)
			r, err := transformer.Transform([]byte(`Aug 21 03:03:04 mail dog: Useless Payload`))
			So(err, ShouldBeNil)
			So(r.Header.Host, ShouldEqual, "mail")
			So(r.Time, ShouldResemble, testutil.MustParseTime(`2000-08-21 03:03:04 +0000`))
		})

		Convey("Default, just return the line, use current year as the passed one is zero", func() {
			builder, err := Get("default", 0)
			So(err, ShouldBeNil)
			transformer, err := builder()
			So(err, ShouldBeNil)
			r, err := transformer.Transform([]byte(`Aug 21 03:03:04 mail dog: Useless Payload`))
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
			_, err := transformer.Transform([]byte(`9898789`))
			So(err, ShouldNotBeNil)
		})

		Convey("Fails invalid time format", func() {
			_, err := transformer.Transform([]byte(`lalala Mar  6 07:08:59 host postfix/qmgr[28829]: A1E1E1880093: removed`))
			So(err, ShouldNotBeNil)
		})

		Convey("Fails invalid line", func() {
			_, err := transformer.Transform([]byte(`2021-03-06T06:09:00.798Z invalid line`))
			So(err, ShouldNotBeNil)
		})

		Convey("Fails ending with space", func() {
			_, err := transformer.Transform([]byte(`2021-03-06T06:09:00.798Z`))
			So(err, ShouldNotBeNil)
		})

		Convey("Succeeds", func() {
			r, err := transformer.Transform([]byte(`2021-03-06T06:09:00.000Z Mar  6 07:08:59 host postfix/qmgr[28829]: A1E1E1880093: removed`))
			So(err, ShouldBeNil)
			So(r.Time, ShouldResemble, testutil.MustParseTime(`2021-03-06 06:09:00 +0000`))
			So(r.Header.Host, ShouldEqual, "host")
		})

		Convey("Succeeds, docker logs default format", func() {
			r, err := transformer.Transform([]byte(`2021-03-08T07:21:23.496826493Z Mar  8 07:21:23 mail postfix/anvil[2990]: statistics: max cache size 2 at Mar  8 07:18:01`))
			So(err, ShouldBeNil)
			So(r.Time, ShouldResemble, time.Date(2021, time.March, 8, 7, 21, 23, 496826493, time.UTC))
			So(r.Header.Host, ShouldEqual, "mail")
		})

	})
}
