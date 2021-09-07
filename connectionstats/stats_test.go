// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package connectionstats

import (
	"context"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/util/postfixutil"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"strings"
	"testing"
	"time"
)

func init() {
	lmsqlite3.Initialize(lmsqlite3.Options{})
}

func TestSmtpConnectionStats(t *testing.T) {
	Convey("Smtp Connection Stats", t, func() {
		db, closeConn := testutil.TempDBConnectionMigrated(t, "connections")
		defer closeConn()

		stats, err := New(db)
		So(err, ShouldBeNil)

		{
			mostRecentTime, err := stats.MostRecentLogTime()
			So(err, ShouldBeNil)
			So(mostRecentTime, ShouldResemble, time.Time{})
		}

		pub := stats.Publisher()
		done, cancel := stats.Run()

		postfixutil.ReadFromTestReader(strings.NewReader(`
Jul 13 17:41:40 mail postfix/smtpd[26098]: disconnect from unknown[11.22.33.44] ehlo=1 auth=8/14 mail=1 rcpt=0/1 data=0/1 rset=1 commands=3/19
Aug 28 20:12:52 mx postfix/smtps/smtpd[8377]: connect from unknown[1002:1712:4e2b:d061:5dff:19f:c85f:a48f]
Sep  1 05:23:56 mail postfix/smtps/smtpd[11962]: connect from unknown[unknown]
Sep  1 05:23:56 mail postfix/smtps/smtpd[11962]: SSL_accept error from unknown[unknown]: Connection reset by peer
Sep  1 05:23:56 mail postfix/smtps/smtpd[11962]: lost connection after CONNECT from unknown[unknown]
Sep  1 05:23:56 mail postfix/smtps/smtpd[11962]: disconnect from unknown[unknown] ehlo=1 auth=0/1 commands=1/2
Sep  3 10:40:57 mail postfix/smtpd[9715]: disconnect from localhost[127.0.0.1] ehlo=1 mail=1 rcpt=1 data=1 quit=1 commands=5
Sep  3 10:40:57 mail postfix/smtpd[9715]: disconnect from example.com[22.33.44.55] ehlo=1 auth=1 mail=1 rcpt=1 data=1 quit=1 commands=6
		`), pub, 2020)

		cancel()
		So(done(), ShouldBeNil)

		{
			mostRecentTime, err := stats.MostRecentLogTime()
			So(err, ShouldBeNil)
			So(mostRecentTime, ShouldResemble, timeutil.MustParseTime(`2020-09-03 10:40:57 +0000`))
		}

		var pool *dbconn.RoPool = stats.ConnPool()

		conn, release := pool.Acquire()
		defer release()

		rows, err := conn.Query(`
			select
				connections.disconnection_ts, lm_ip_to_string(connections.ip), commands.cmd, commands.success, commands.total
			from
				connections join commands on commands.connection_id = connections.id
			order by
				connections.id asc, commands.cmd asc
		`)

		So(err, ShouldBeNil)

		defer rows.Close()

		type result struct {
			time    time.Time
			ip      string
			cmd     Command
			success int
			total   int
		}

		var results []result

		for rows.Next() {
			var (
				ts      int64
				ip      string
				cmd     int
				success int
				total   int
			)

			err = rows.Scan(&ts, &ip, &cmd, &success, &total)
			So(err, ShouldBeNil)

			result := result{time: time.Unix(ts, 0).In(time.UTC), ip: ip, cmd: Command(cmd), success: success, total: total}
			results = append(results, result)
		}

		expectedTime1 := timeutil.MustParseTime(`2020-07-13 17:41:40 +0000`)
		expectedIP1 := "11.22.33.44"

		expectedTime2 := timeutil.MustParseTime(`2020-09-03 10:40:57 +0000`)
		expectedIP2 := "22.33.44.55"

		So(results, ShouldResemble, []result{
			{time: expectedTime1, ip: expectedIP1, cmd: AuthCommand, success: 8, total: 14},
			{time: expectedTime1, ip: expectedIP1, cmd: DataCommand, success: 0, total: 1},
			{time: expectedTime1, ip: expectedIP1, cmd: EhloCommand, success: 1, total: 1},
			{time: expectedTime1, ip: expectedIP1, cmd: MailCommand, success: 1, total: 1},
			{time: expectedTime1, ip: expectedIP1, cmd: RcptCommand, success: 0, total: 1},
			{time: expectedTime1, ip: expectedIP1, cmd: RsetCommand, success: 1, total: 1},

			{time: expectedTime2, ip: expectedIP2, cmd: AuthCommand, success: 1, total: 1},
			{time: expectedTime2, ip: expectedIP2, cmd: DataCommand, success: 1, total: 1},
			{time: expectedTime2, ip: expectedIP2, cmd: EhloCommand, success: 1, total: 1},
			{time: expectedTime2, ip: expectedIP2, cmd: MailCommand, success: 1, total: 1},
			{time: expectedTime2, ip: expectedIP2, cmd: QuitCommand, success: 1, total: 1},
			{time: expectedTime2, ip: expectedIP2, cmd: RcptCommand, success: 1, total: 1},
		})
	})
}

func TestSmtpConnectionAccessor(t *testing.T) {
	Convey("Smtp Connection Stats", t, func() {
		db, closeConn := testutil.TempDBConnectionMigrated(t, "connections")
		defer closeConn()

		stats, err := New(db)
		So(err, ShouldBeNil)

		pub := stats.Publisher()

		var pool *dbconn.RoPool = stats.ConnPool()

		accessor, err := NewAccessor(pool)
		So(err, ShouldBeNil)

		{
			// Before any logs, nothing should be returned
			attempts, err := accessor.FetchAuthAttempts(context.Background(), timeutil.TimeInterval{
				From: timeutil.MustParseTime(`2000-01-01 00:00:00 +0000`),
				To:   timeutil.MustParseTime(`4000-01-01 00:00:00 +0000`),
			})

			So(err, ShouldBeNil)

			So(attempts.IPs, ShouldResemble, []string{})
			So(attempts.Attempts, ShouldResemble, []AttemptDesc{})
		}

		done, cancel := stats.Run()

		postfixutil.ReadFromTestReader(strings.NewReader(`
Jan 10 17:41:40 mail postfix/smtpd[1234]: disconnect from unknown[4.3.2.1] ehlo=1 auth=1 mail=1 rcpt=1 data=1 rset=1 commands=6
Jul 13 17:41:40 mail postfix/smtpd[26098]: disconnect from unknown[11.22.33.44] ehlo=1 auth=8/14 mail=1 rcpt=0/1 data=0/1 rset=1 commands=3/19
Sep  3 10:40:57 mail postfix/smtpd[8377]: disconnect from lalala.com[1002:1712:4e2b:d061:5dff:19f:c85f:a48f] ehlo=1 auth=0/1 commands=1/2
Sep  4 10:40:57 mail postfix/smtpd[9715]: disconnect from example.com[22.33.44.55] ehlo=1 auth=1 mail=1 rcpt=1 data=1 quit=1 commands=6
Dec 30 10:40:57 mail postfix/smtpd[4567]: disconnect from example.com[1.2.3.4] ehlo=1 auth=1 mail=1 rcpt=1 data=1 quit=1 commands=6
		`), pub, 2020)

		cancel()
		done()

		{
			// After the logs, we should have some results, and we get a subset of it. The first and last log lines are out
			attempts, err := accessor.FetchAuthAttempts(context.Background(), timeutil.TimeInterval{
				From: timeutil.MustParseTime(`2020-07-01 00:00:00 +0000`),
				To:   timeutil.MustParseTime(`2020-10-01 00:00:00 +0000`),
			})

			So(err, ShouldBeNil)

			So(attempts.IPs, ShouldResemble, []string{"1002:1712:4e2b:d061:5dff:19f:c85f:a48f", "11.22.33.44", "22.33.44.55"})
			So(attempts.Attempts, ShouldResemble, []AttemptDesc{
				{Time: timeutil.MustParseTime(`2020-07-13 17:41:40 +0000`).Unix(), IPIndex: 1, Status: "suspicious"},
				{Time: timeutil.MustParseTime(`2020-09-03 10:40:57 +0000`).Unix(), IPIndex: 0, Status: "failed"},
				{Time: timeutil.MustParseTime(`2020-09-04 10:40:57 +0000`).Unix(), IPIndex: 2, Status: "ok"},
			})
		}
	})
}
