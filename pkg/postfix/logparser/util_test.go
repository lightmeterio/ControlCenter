// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0

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

	Convey("Regression: Short out of order are supported", t, func() {
		// Sometimes some log lines are out of order for a few seconds (it happens especially when they are logged by different processes)
		// and we should support such cases, not bumping the year.
		// One example is the following lines:
		/*
			Dec 17 10:42:27 sm02 postfix/cleanup[27439]: E6E98DC28ED: message-id=11111
			Dec 17 10:42:27 sm02 opendkim[94828]: E6E98DC28ED: no signing table match for 'some.email@example.com'
			Dec 17 10:42:27 sm02 postfix/dkimmilter/smtpd[27443]: disconnect from localhost[127.0.0.1] ehlo=1 mail=1 rcpt=1 data=1 quit=1 commands=5
			Dec 17 10:42:28 sm02 amavis[13720]: (13720-05) 1PuXuLcNJCsB FWD from <sender@sender.example.com> -> <recipient@recipient.example.com>, BODY=7BIT 250 2.0.0 from MTA(smtp:[127.0.0.1]:10030): 250 2.0.0 Ok: queued as E6E98DC28ED
			Dec 17 10:42:27 sm02 postfix/qmgr[95074]: E6E98DC28ED: from=<sender@sender.example.com>, size=55485, nrcpt=1 (queue active)
			Dec 17 10:42:28 sm02 amavis[13720]: (13720-05) Passed CLEAN {RelayedOutbound}, ORIGINATING LOCAL [118.69.64.170]:61810 [118.69.64.170] <sender@sender.example.com> -> <recipient@recipient.example.com>, Queue-ID: D4280DC299D, Message-ID: <11111>, mail_id: 1PuXuLcNJCsB, Hits: -, size: 55031, queued_as: E6E98DC28ED, 277 ms
		*/

		initialTime := time.Date(2000, time.January, 1, 0, 0, 0, 0, tz)
		yearOffset := 0

		c := NewTimeConverter(initialTime, func(int, Time, Time) { yearOffset++ })

		c.Convert(Time{Month: time.December, Day: 17, Hour: 10, Minute: 42, Second: 27})
		c.Convert(Time{Month: time.December, Day: 17, Hour: 10, Minute: 42, Second: 27})
		c.Convert(Time{Month: time.December, Day: 17, Hour: 10, Minute: 42, Second: 28})
		c.Convert(Time{Month: time.December, Day: 17, Hour: 10, Minute: 42, Second: 27})
		c.Convert(Time{Month: time.December, Day: 17, Hour: 10, Minute: 42, Second: 28})
		c.Convert(Time{Month: time.December, Day: 17, Hour: 10, Minute: 42, Second: 27})
		c.Convert(Time{Month: time.December, Day: 17, Hour: 10, Minute: 42, Second: 28})
		c.Convert(Time{Month: time.December, Day: 17, Hour: 10, Minute: 42, Second: 27})
		c.Convert(Time{Month: time.December, Day: 17, Hour: 10, Minute: 42, Second: 28})
		c.Convert(Time{Month: time.December, Day: 17, Hour: 10, Minute: 42, Second: 28})
		c.Convert(Time{Month: time.December, Day: 17, Hour: 10, Minute: 42, Second: 29})
		c.Convert(Time{Month: time.December, Day: 17, Hour: 10, Minute: 42, Second: 30})
		c.Convert(Time{Month: time.December, Day: 17, Hour: 10, Minute: 42, Second: 29})
		c.Convert(Time{Month: time.December, Day: 17, Hour: 10, Minute: 42, Second: 29})
		c.Convert(Time{Month: time.December, Day: 17, Hour: 10, Minute: 42, Second: 29})
		c.Convert(Time{Month: time.December, Day: 17, Hour: 10, Minute: 42, Second: 30})
		c.Convert(Time{Month: time.December, Day: 17, Hour: 10, Minute: 42, Second: 31})

		So(c.year, ShouldEqual, 2000)
	})

	Convey("Regression: when daylight saving time ends, time moves one hour backward (damn syslog!)", t, func() {
		/*
			syslog moves time backwards when daylight saving time finishes (in places where it's used), what can mess up with tracking year...
			Like in the example:

			Oct 25 02:58:39 ucs postfix/smtpd[24944]: disconnect from unknown[11.172.110.172] ehlo=1 auth=0/1 rset=1 commands=2/3
			Oct 25 02:59:46 ucs postfix/smtpd[24944]: connect from h-2dc03ed8c98dd0.h-038860858e95dc[209.170.217.165]
			Oct 25 02:59:46 ucs postfix/smtpd[24944]: warning: SASL authentication failure: cannot connect to saslauthd server: Connection refused
			Oct 25 02:59:46 ucs postfix/smtpd[24944]: warning: h-2dc03ed8c98dd0.h-038860858e95dc[209.170.217.165]: SASL LOGIN authentication failed: generic failure
			Oct 25 02:59:46 ucs postfix/smtpd[24944]: disconnect from h-2dc03ed8c98dd0.h-038860858e95dc[209.170.217.165] ehlo=1 auth=0/1 quit=1 commands=2/3
			Oct 25 02:00:03 ucs postfix/smtpd[24944]: connect from unknown[11.172.110.172]
			Oct 25 02:00:34 ucs postfix/smtpd[24944]: warning: SASL authentication failure: cannot connect to saslauthd server: Connection refused
			Oct 25 02:00:34 ucs postfix/smtpd[24944]: warning: unknown[11.172.110.172]: SASL LOGIN authentication failed: generic failure
		*/

		initialTime := time.Date(2000, time.January, 1, 0, 0, 0, 0, tz)
		yearOffset := 0

		c := NewTimeConverter(initialTime, func(int, Time, Time) { yearOffset++ })

		c.Convert(Time{Month: time.October, Day: 25, Hour: 2, Minute: 59, Second: 45})
		c.Convert(Time{Month: time.October, Day: 25, Hour: 2, Minute: 59, Second: 46})
		// time is set one hour backwards here. Year should not change!
		c.Convert(Time{Month: time.October, Day: 25, Hour: 2, Minute: 00, Second: 03})
		c.Convert(Time{Month: time.October, Day: 25, Hour: 2, Minute: 00, Second: 34})

		So(c.year, ShouldEqual, 2000)
	})
}
