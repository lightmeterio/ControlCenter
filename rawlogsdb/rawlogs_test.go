// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package rawlogsdb

import (
	"bytes"
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

func TestFetchingLogLines(t *testing.T) {
	Convey("Test Fetching Log Lines", t, func() {
		rawLogs, pub, pool, closeConn := buildContext(t)
		defer closeConn()

		done, cancel := runner.Run(rawLogs)

		{
			// No logs are available, time is unavailable
			sum, err := MostRecentLogTimeAndSum(context.Background(), pool)
			So(err, ShouldBeNil)
			So(sum.Time.IsZero(), ShouldBeTrue)
			So(sum.Sum, ShouldBeNil)
		}

		postfixutil.ReadFromTestFile("../test_files/postfix_logs/individual_files/4_lost_queue.log", pub, 2020, &timeutil.FakeClock{Time: timeutil.MustParseTime(`2020-12-31 00:00:00 +0000`)})

		cancel()
		So(done(), ShouldBeNil)

		{
			sum, err := MostRecentLogTimeAndSum(context.Background(), pool)
			So(err, ShouldBeNil)
			So(sum.Time, ShouldResemble, timeutil.MustParseTime(`2020-02-04 09:29:33 +0000`))
			So(sum.Sum, ShouldNotBeNil)
			So(*sum.Sum, ShouldResemble, postfix.ComputeChecksum(postfix.NewHasher(), `Feb  4 09:29:33 mail postfix/qmgr[964]: 027BD2C77B20: removed`))
		}

		interval := timeutil.TimeInterval{
			From: timeutil.MustParseTime(`2020-02-04 09:29:27 +0000`),
			To:   timeutil.MustParseTime(`2020-02-04 09:29:29 +0000`),
		}

		count, err := CountLogLinesInInterval(context.Background(), pool, interval)
		So(err, ShouldBeNil)
		So(count, ShouldEqual, 40)

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

		Convey("Cleaning should not remove any lines if no lines are available", func() {
			// nothing is removed
			rawLogs.Actions <- makeCleanAction(5*time.Second, 40)
			cancel()
			So(done(), ShouldBeNil)

			interval := timeutil.TimeInterval{
				From: timeutil.MustParseTime(`2000-01-01 00:00:00 +0000`),
				To:   timeutil.MustParseTime(`4000-01-01 00:00:00 +0000`),
			}

			r, err := FetchLogsInInterval(context.Background(), pool, interval, 1000, 0)
			So(err, ShouldBeNil)
			So(len(r.Content), ShouldEqual, 0)
		})

		Convey("With existing logs", func() {
			postfixutil.ReadFromTestFile("../test_files/postfix_logs/individual_files/4_lost_queue.log", pub, 2020, &timeutil.FakeClock{Time: timeutil.MustParseTime(`2020-12-31 00:00:00 +0000`)})

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
	})
}

func TestFetchingRawContent(t *testing.T) {
	Convey("Test Fetching Raw Content", t, func() {
		rawLogs, pub, pool, closeConn := buildContext(t)
		defer closeConn()

		done, cancel := runner.Run(rawLogs)

		postfixutil.ReadFromTestFile("../test_files/postfix_logs/individual_files/5_zimbra_amavisd.log", pub, 2020, &timeutil.FakeClock{Time: timeutil.MustParseTime(`2020-12-31 00:00:00 +0000`)})

		cancel()
		So(done(), ShouldBeNil)

		interval := timeutil.TimeInterval{
			From: timeutil.MustParseTime(`2020-12-17 06:29:07 +0000`),
			To:   timeutil.MustParseTime(`2020-12-17 06:29:10 +0000`),
		}

		var buffer bytes.Buffer

		err := FetchLogsInIntervalToWriter(context.Background(), pool, interval, &buffer)
		So(err, ShouldBeNil)

		So(buffer.String(), ShouldEqual, `Dec 17 06:29:07 sm02 postfix/amavisd/smtpd[115286]: connect from localhost[127.0.0.1]
Dec 17 06:29:07 sm02 postfix/amavisd/smtpd[115286]: 33DE0DC24C3: client=localhost[127.0.0.1]
Dec 17 06:29:07 sm02 postfix/cleanup[115121]: 33DE0DC24C3: message-id=<00000messageid00000@msgid.example.com>
Dec 17 06:29:07 sm02 postfix/amavisd/smtpd[115286]: disconnect from localhost[127.0.0.1] ehlo=1 mail=1 rcpt=1 data=1 quit=1 commands=5
Dec 17 06:29:07 sm02 amavis[11202]: (11202-06) T0hCuLOMViWI FWD from <sender@sender.example.com> -> <recipient@recipient.example.com>, BODY=7BIT 250 2.0.0 from MTA(smtp:[127.0.0.1]:10025): 250 2.0.0 Ok: queued as 33DE0DC24C3
Dec 17 06:29:07 sm02 postfix/qmgr[95074]: 33DE0DC24C3: from=<sender@sender.example.com>, size=344822, nrcpt=1 (queue active)
Dec 17 06:29:07 sm02 amavis[11202]: (11202-06) Passed CLEAN {RelayedOutbound}, ORIGINATING_POST/MYNETS LOCAL [127.0.0.1]:59654 [11.22.33.44] <sender@sender.example.com> -> <recipient@recipient.example.com>, Queue-ID: 218BEDC28EC, Message-ID: <00000messageid00000@msgid.example.com>, mail_id: T0hCuLOMViWI, Hits: -2.789, size: 344431, queued_as: 33DE0DC24C3, 9115 ms
Dec 17 06:29:07 sm02 amavis[11202]: (11202-06) TIMING-SA [total 8782 ms, cpu 3158 ms] - parse: 22 (0.2%), extract_message_metadata: 163 (1.9%), get_uri_detail_list: 12 (0.1%), tests_pri_-1000: 9 (0.1%), tests_pri_-950: 1.42 (0.0%), tests_pri_-900: 1.13 (0.0%), tests_pri_-90: 0.98 (0.0%), tests_pri_0: 2729 (31.1%), check_spf: 0.43 (0.0%), check_dkim_adsp: 8 (0.1%), tests_pri_10: 4547 (51.8%), check_dcc: 4536 (51.7%), tests_pri_20: 948 (10.8%), check_razor2: 935 (10.6%), tests_pri_30: 337 (3.8%), check_pyzor: 334 (3.8%), tests_pri_500: 13 (0.1%), get_report: 1.01 (0.0%)
Dec 17 06:29:07 sm02 amavis[11202]: (11202-06) size: 344431, TIMING [total 9119 ms, cpu 3305 ms, AM-cpu 147 ms, SA-cpu 3158 ms] - SMTP greeting: 2.3 (0%)0, SMTP EHLO: 1.0 (0%)0, SMTP pre-MAIL: 0.8 (0%)0, lookup_ldap: 5 (0%)0, SMTP pre-DATA-flush: 0.8 (0%)0, SMTP DATA: 40 (0%)1, check_init: 0.3 (0%)1, digest_hdr: 1.6 (0%)1, digest_body_dkim: 3.2 (0%)1, collect_info: 3.3 (0%)1, mime_decode: 40 (0%)1, get-file-type5: 47 (1%)2, parts_decode: 0.3 (0%)2, check_header: 0.6 (0%)2, spam-wb-list: 3.6 (0%)2, SA parse: 23 (0%)2, SA check: 8757 (96%)98, decide_mail_destiny: 15 (0%)98, notif-quar: 0.4 (0%)98, fwd-connect: 50 (1%)99, fwd-mail-pip: 14 (0%)99, fwd-rcpt-pip: 0.2 (0%)99, fwd-data-chkpnt: 0.1 (0%)99, write-header: 0.7 (0%)99, fwd-data-contents: 14 (0%)99, fwd-end-chkpnt: 82 (1%)100, prepare-dsn: 1.0 (0%)100, report: 1.9 (0%)100, main_log_entry: 6 (0%)100, update_snmp: 0.6 (0%)100, SMTP pre-response: 0.2 (0%)100, SMTP response: 0.6 (0%)100, unlink-5-files: 0.4 (0%)100, rundown: 1.2 (0%)100
Dec 17 06:29:07 sm02 amavis[11202]: (11202-06) size: 344431, RUSAGE minflt=15507+3931, majflt=0+0, nswap=0+0, inblock=0+0, oublock=8+8, msgsnd=0+0, msgrcv=0+0, nsignals=0+0, nvcsw=27+19, nivcsw=64+7, maxrss=185300+179328, ixrss=0+0, idrss=0+0, isrss=0+0, utime=3.151+0.069, stime=0.051+0.034
Dec 17 06:29:07 sm02 postfix/smtp[115122]: 218BEDC28EC: to=<recipient@recipient.example.com>, relay=127.0.0.1[127.0.0.1]:10032, delay=9.2, delays=0.06/0.01/0/9.1, dsn=2.0.0, status=sent (250 2.0.0 from MTA(smtp:[127.0.0.1]:10025): 250 2.0.0 Ok: queued as 33DE0DC24C3)
Dec 17 06:29:07 sm02 postfix/qmgr[95074]: 218BEDC28EC: removed
Dec 17 06:29:08 sm02 /postfix-script[115413]: the Postfix mail system is running: PID: 95072
Dec 17 06:29:10 sm02 zmconfigd[91758]: Watchdog: service antivirus status is OK.
Dec 17 06:29:10 sm02 zmconfigd[91758]: All rewrite threads completed in 0.00 sec
Dec 17 06:29:10 sm02 zmconfigd[91758]: All restarts completed in 0.00 sec
Dec 17 06:29:10 sm02 slapd[91734]: slap_queue_csn: queueing 0x3311e00 20201216232910.480418Z#000000#000#000000
Dec 17 06:29:10 sm02 slapd[91734]: slap_graduate_commit_csn: removing 0x3311e00 20201216232910.480418Z#000000#000#000000
`)

	})
}
