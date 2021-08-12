// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package connectionstats

import (
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
		ws, clearWs := testutil.TempDir(t)
		defer clearWs()

		stats, err := New(ws)
		So(err, ShouldBeNil)

		pub := stats.Publisher()
		done, cancel := stats.Run()

		postfixutil.ReadFromTestReader(strings.NewReader(`
Jul 13 17:41:40 mail postfix/smtpd[26098]: disconnect from unknown[11.22.33.44] ehlo=1 auth=8/14 mail=1 rcpt=0/1 data=0/1 rset=1 commands=3/19
Aug 28 20:12:52 mx postfix/smtps/smtpd[8377]: connect from unknown[1002:1712:4e2b:d061:5dff:19f:c85f:a48f]
Sep  3 10:40:57 mail postfix/smtpd[9715]: disconnect from localhost[127.0.0.1] ehlo=1 mail=1 rcpt=1 data=1 quit=1 commands=5
Sep  3 10:40:57 mail postfix/smtpd[9715]: disconnect from example.com[22.33.44.55] ehlo=1 auth=1 mail=1 rcpt=1 data=1 quit=1 commands=6
		`), pub, 2020)

		cancel()
		done()

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
