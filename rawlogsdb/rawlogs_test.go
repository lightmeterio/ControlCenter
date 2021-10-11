// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package rawlogsdb

import (
	"context"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/pkg/postfix"
	"gitlab.com/lightmeter/controlcenter/pkg/runner"
	"gitlab.com/lightmeter/controlcenter/util/postfixutil"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"testing"
	"time"
)

func init() {
	lmsqlite3.Initialize(lmsqlite3.Options{})
}

func buildContext(t *testing.T) (*DB, postfix.Publisher, *dbconn.RoPool, func()) {
	db, closeConn := testutil.TempDBConnectionMigrated(t, "rawlogs")

	r, err := New(db.RwConn)
	So(err, ShouldBeNil)

	pub := r.Publisher()

	return r, pub, db.RoConnPool, func() {
		closeConn()
	}
}

func TestRawLogs(t *testing.T) {
	Convey("Test Raw Logs", t, func() {
		rawLogs, pub, pool, closeConn := buildContext(t)
		defer closeConn()

		done, cancel := runner.Run(rawLogs)

		postfixutil.ReadFromTestFile("../test_files/postfix_logs/individual_files/4_lost_queue.log", pub, 2020)

		cancel()
		So(done(), ShouldBeNil)

		interval := timeutil.TimeInterval{
			From: timeutil.MustParseTime(`2020-02-04 09:29:27 +0000`),
			To:   timeutil.MustParseTime(`2020-02-04 09:29:29 +0000`),
		}

		// first page
		{
			r, err := FetchLogsInInterval(context.Background(), pool, interval, 10, 0)
			So(err, ShouldBeNil)
			So(len(r.Content), ShouldEqual, 10)
			So(r.Cursor, ShouldEqual, 34)
		}

		// second page, with cursor from the previous execution, and fetch more lines
		{
			r, err := FetchLogsInInterval(context.Background(), pool, interval, 20, 34)
			So(err, ShouldBeNil)
			So(len(r.Content), ShouldEqual, 20)
			So(r.Cursor, ShouldEqual, 54)
		}

		// third page, fetch less than the page size
		{
			r, err := FetchLogsInInterval(context.Background(), pool, interval, 20, 54)
			So(err, ShouldBeNil)
			So(len(r.Content), ShouldEqual, 10)
			So(r.Cursor, ShouldEqual, 64)
		}

		// nothing in the 4th page
		{
			r, err := FetchLogsInInterval(context.Background(), pool, interval, 20, 64)
			So(err, ShouldBeNil)
			So(len(r.Content), ShouldEqual, 0)
			So(r.Cursor, ShouldEqual, 0)
		}
	})
}

func TestDeleteLogs(t *testing.T) {
	Convey("Test Deleting Logs", t, func() {
		rawLogs, pub, pool, closeConn := buildContext(t)
		defer closeConn()

		done, cancel := runner.Run(rawLogs)

		postfixutil.ReadFromTestFile("../test_files/postfix_logs/individual_files/4_lost_queue.log", pub, 2020)

		// remove the first items
		rawLogs.Actions <- makeCleanAction(5*time.Second, 40)
		rawLogs.Actions <- makeCleanAction(5*time.Second, 20)
		rawLogs.Actions <- makeCleanAction(5*time.Second, 10)

		cancel()
		So(done(), ShouldBeNil)

		interval := timeutil.TimeInterval{
			From: timeutil.MustParseTime(`2000-01-01 00:00:00 +0000`),
			To:   timeutil.MustParseTime(`4000-01-01 00:00:00 +0000`),
		}

		r, err := FetchLogsInInterval(context.Background(), pool, interval, 1000, 0)
		So(err, ShouldBeNil)
		So(len(r.Content), ShouldEqual, 3)
		So(r.Cursor, ShouldEqual, 66)
		So(r.Content, ShouldResemble, []ContentRow{
			{
				Timestamp: timeutil.MustParseTime(`2020-02-04 09:29:29 +0000`).Unix(),
				Content:   `Feb  4 09:29:29 mail postfix/smtp[6655]: Anonymous TLS connection established to balzers.recipient.example.com[217.173.226.42]:25: TLSv1.2 with cipher ADH-AES256-GCM-SHA384 (256/256 bits)`,
			},
			{
				Timestamp: timeutil.MustParseTime(`2020-02-04 09:29:33 +0000`).Unix(),
				Content:   `Feb  4 09:29:33 mail postfix/smtp[6655]: 027BD2C77B20: to=<recipient1@recipient.example.com>, relay=balzers.recipient.example.com[217.173.226.42]:25, delay=6.9, delays=0.01/0.02/2.9/4, dsn=2.0.0, status=sent (250 2.0.0 from MTA(smtp:[127.0.0.1]:10025): 250 2.0.0 Ok: queued as ADCC76373)`,
			},
			{
				Timestamp: timeutil.MustParseTime(`2020-02-04 09:29:33 +0000`).Unix(),
				Content:   `Feb  4 09:29:33 mail postfix/qmgr[964]: 027BD2C77B20: removed`,
			},
		})
	})
}
