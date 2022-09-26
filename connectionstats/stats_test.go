// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package connectionstats

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/pkg/postfix"
	"gitlab.com/lightmeter/controlcenter/pkg/runner"
	"gitlab.com/lightmeter/controlcenter/util/postfixutil"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
)

func init() {
	lmsqlite3.Initialize(lmsqlite3.Options{})
}

func TestSmtpConnectionStats(t *testing.T) {
	Convey("Smtp Connection Stats", t, func() {
		stats, _, pub, pool, closeConn := buildContext(t)
		defer closeConn()

		{
			mostRecentTime, err := stats.MostRecentLogTime()
			So(err, ShouldBeNil)
			So(mostRecentTime, ShouldResemble, time.Time{})
		}

		done, cancel := runner.Run(stats)

		postfixutil.ReadFromTestReader(strings.NewReader(`
Jul 13 17:41:40 mail postfix/smtpd[26098]: disconnect from unknown[11.22.33.44] ehlo=1 auth=8/14 mail=1 rcpt=0/1 data=0/1 rset=1 commands=3/19
Aug 28 20:12:52 mx postfix/smtps/smtpd[8377]: connect from unknown[1002:1712:4e2b:d061:5dff:19f:c85f:a48f]
Sep  1 05:23:56 mail postfix/smtps/smtpd[11962]: connect from unknown[unknown]
Sep  1 05:23:56 mail postfix/smtps/smtpd[11962]: SSL_accept error from unknown[unknown]: Connection reset by peer
Sep  1 05:23:56 mail postfix/smtps/smtpd[11962]: lost connection after CONNECT from unknown[unknown]
Sep  1 05:23:56 mail postfix/smtps/smtpd[11962]: disconnect from unknown[unknown] ehlo=1 auth=0/1 commands=1/2
Sep  3 10:40:57 mail postfix/smtpd[9715]: disconnect from localhost[127.0.0.1] ehlo=1 mail=1 rcpt=1 data=1 quit=1 commands=5
Sep  3 10:40:57 mail postfix/smtpd[9715]: disconnect from example.com[22.33.44.55] ehlo=1 auth=1 mail=1 rcpt=1 data=1 quit=1 commands=6
Sep  3 11:30:51 mail dovecot: auth: passwd-file(alice,1.2.3.4): unknown user (SHA1 of given password: 011c94)
		`), pub, 2020, &timeutil.FakeClock{Time: timeutil.MustParseTime(`2020-10-15 00:00:00 +0000`)})

		cancel()
		So(done(), ShouldBeNil)

		{
			mostRecentTime, err := stats.MostRecentLogTime()
			So(err, ShouldBeNil)
			So(mostRecentTime, ShouldResemble, timeutil.MustParseTime(`2020-09-03 11:30:51 +0000`))
		}

		conn, release := pool.Acquire()
		defer release()

		rows, err := conn.Query(`
			select
				connections.disconnection_ts, connections.protocol, lm_ip_to_string(connections.ip), commands.cmd, commands.success, commands.total
			from
				connections join commands on commands.connection_id = connections.id
			order by
				connections.id asc, commands.cmd asc
		`)

		So(err, ShouldBeNil)

		defer rows.Close()

		type result struct {
			time     time.Time
			protocol Protocol
			ip       string
			cmd      Command
			success  int
			total    int
		}

		var results []result

		for rows.Next() {
			var (
				ts       int64
				protocol Protocol
				ip       string
				cmd      int
				success  int
				total    int
			)

			err = rows.Scan(&ts, &protocol, &ip, &cmd, &success, &total)
			So(err, ShouldBeNil)

			result := result{time: time.Unix(ts, 0).In(time.UTC), protocol: protocol, ip: ip, cmd: Command(cmd), success: success, total: total}
			results = append(results, result)
		}

		expectedTime1 := timeutil.MustParseTime(`2020-07-13 17:41:40 +0000`)
		expectedIP1 := "11.22.33.44"

		expectedTime2 := timeutil.MustParseTime(`2020-09-03 10:40:57 +0000`)
		expectedIP2 := "22.33.44.55"

		expectedTime3 := timeutil.MustParseTime(`2020-09-03 11:30:51 +0000`)
		expectedIP3 := "1.2.3.4"

		So(results, ShouldResemble, []result{
			{time: expectedTime1, ip: expectedIP1, protocol: ProtocolSMTP, cmd: AuthCommand, success: 8, total: 14},
			{time: expectedTime1, ip: expectedIP1, protocol: ProtocolSMTP, cmd: DataCommand, success: 0, total: 1},
			{time: expectedTime1, ip: expectedIP1, protocol: ProtocolSMTP, cmd: EhloCommand, success: 1, total: 1},
			{time: expectedTime1, ip: expectedIP1, protocol: ProtocolSMTP, cmd: MailCommand, success: 1, total: 1},
			{time: expectedTime1, ip: expectedIP1, protocol: ProtocolSMTP, cmd: RcptCommand, success: 0, total: 1},
			{time: expectedTime1, ip: expectedIP1, protocol: ProtocolSMTP, cmd: RsetCommand, success: 1, total: 1},

			{time: expectedTime2, ip: expectedIP2, protocol: ProtocolSMTP, cmd: AuthCommand, success: 1, total: 1},
			{time: expectedTime2, ip: expectedIP2, protocol: ProtocolSMTP, cmd: DataCommand, success: 1, total: 1},
			{time: expectedTime2, ip: expectedIP2, protocol: ProtocolSMTP, cmd: EhloCommand, success: 1, total: 1},
			{time: expectedTime2, ip: expectedIP2, protocol: ProtocolSMTP, cmd: MailCommand, success: 1, total: 1},
			{time: expectedTime2, ip: expectedIP2, protocol: ProtocolSMTP, cmd: QuitCommand, success: 1, total: 1},
			{time: expectedTime2, ip: expectedIP2, protocol: ProtocolSMTP, cmd: RcptCommand, success: 1, total: 1},

			{time: expectedTime3, ip: expectedIP3, protocol: ProtocolIMAP, cmd: DovecotAuthCommand, success: 0, total: 1},
		})
	})
}

func TestJSONSerialization(t *testing.T) {
	Convey("JSON serialization", t, func() {
		a := AccessResult{
			IPs: []string{"1.1.1.1", "2.2.2.2"},
			Attempts: []AttemptDesc{
				{Time: timeutil.MustParseTime(`2000-01-01 00:00:00 +0000`).Unix(), IPIndex: 1, Status: "blocked", Protocol: ProtocolIMAP},
				{Time: timeutil.MustParseTime(`2000-01-01 00:00:01 +0000`).Unix(), IPIndex: 0, Status: "ok", Protocol: ProtocolSMTP},
			},
		}

		j, err := json.Marshal(a)
		So(err, ShouldBeNil)

		{
			var decoded interface{}

			err := json.Unmarshal(j, &decoded)
			So(err, ShouldBeNil)

			So(decoded, ShouldResemble, map[string]interface{}{
				"ips": []interface{}{"1.1.1.1", "2.2.2.2"},
				"attempts": []interface{}{
					map[string]interface{}{
						"time":     float64(timeutil.MustParseTime(`2000-01-01 00:00:00 +0000`).Unix()),
						"index":    float64(1),
						"status":   "blocked",
						"protocol": "imap",
					},
					map[string]interface{}{
						"time":     float64(timeutil.MustParseTime(`2000-01-01 00:00:01 +0000`).Unix()),
						"index":    float64(0),
						"status":   "ok",
						"protocol": "smtp",
					},
				},
			})
		}
	})
}

func TestSmtpConnectionAccessor(t *testing.T) {
	Convey("Smtp Connection Stats", t, func() {
		stats, accessor, pub, _, closeConn := buildContext(t)
		defer closeConn()

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

		done, cancel := runner.Run(stats)

		postfixutil.ReadFromTestReader(strings.NewReader(`
Jan 10 17:41:40 mail postfix/smtpd[1234]: disconnect from unknown[4.3.2.1] ehlo=1 auth=1 mail=1 rcpt=1 data=1 rset=1 commands=6
Jul 13 17:41:40 mail postfix/smtpd[26098]: disconnect from unknown[11.22.33.44] ehlo=1 auth=8/14 mail=1 rcpt=0/1 data=0/1 rset=1 commands=3/19
Sep  3 10:40:57 mail postfix/smtpd[8377]: disconnect from lalala.com[1002:1712:4e2b:d061:5dff:19f:c85f:a48f] ehlo=1 auth=0/1 commands=1/2
Sep  4 10:40:57 mail postfix/smtpd[9715]: disconnect from example.com[22.33.44.55] ehlo=1 auth=1 mail=1 rcpt=1 data=1 quit=1 commands=6
Sep  4 11:30:51 mail dovecot: auth: passwd-file(alice,44.44.44.44): unknown user (SHA1 of given password: 011c94)
Sep 18 18:54:59 mail dovecot: auth: policy(maintenance,55.55.55.55): Authentication failure due to policy server refusal
Dec 30 10:40:57 mail postfix/smtpd[4567]: disconnect from example.com[1.2.3.4] ehlo=1 auth=1 mail=1 rcpt=1 data=1 quit=1 commands=6
		`), pub, 2020, &timeutil.FakeClock{Time: timeutil.MustParseTime(`2020-12-31 00:00:00 +0000`)})

		cancel()
		So(done(), ShouldBeNil)

		{
			// After the logs, we should have some results, and we get a subset of it. The first and last log lines are out
			attempts, err := accessor.FetchAuthAttempts(context.Background(), timeutil.TimeInterval{
				From: timeutil.MustParseTime(`2020-07-01 00:00:00 +0000`),
				To:   timeutil.MustParseTime(`2020-10-01 00:00:00 +0000`),
			})

			So(err, ShouldBeNil)

			So(attempts.IPs, ShouldResemble, []string{"1002:1712:4e2b:d061:5dff:19f:c85f:a48f", "11.22.33.44", "22.33.44.55", "44.44.44.44", "55.55.55.55"})
			So(attempts.Attempts, ShouldResemble, []AttemptDesc{
				{Time: timeutil.MustParseTime(`2020-07-13 17:41:40 +0000`).Unix(), IPIndex: 1, Status: "suspicious", Protocol: ProtocolSMTP},
				{Time: timeutil.MustParseTime(`2020-09-03 10:40:57 +0000`).Unix(), IPIndex: 0, Status: "failed", Protocol: ProtocolSMTP},
				{Time: timeutil.MustParseTime(`2020-09-04 10:40:57 +0000`).Unix(), IPIndex: 2, Status: "ok", Protocol: ProtocolSMTP},
				{Time: timeutil.MustParseTime(`2020-09-04 11:30:51 +0000`).Unix(), IPIndex: 3, Status: "failed", Protocol: ProtocolIMAP},
				{Time: timeutil.MustParseTime(`2020-09-18 18:54:59 +0000`).Unix(), IPIndex: 4, Status: "blocked", Protocol: ProtocolIMAP},
			})
		}
	})
}

func buildContext(t *testing.T) (*Stats, *Accessor, postfix.Publisher, *dbconn.RoPool, func()) {
	db, closeConn := testutil.TempDBConnectionMigrated(t, "connections")
	options := Options{RetentionDuration: (time.Hour * 24 * 30 * 3)}
	stats, err := New(db, options)
	So(err, ShouldBeNil)

	pub := stats.Publisher()

	accessor, err := NewAccessor(db.RoConnPool)
	So(err, ShouldBeNil)

	return stats, accessor, pub, accessor.pool, func() {
		closeConn()
	}
}

func TestRemoveOldLogs(t *testing.T) {
	Convey("Remove Old Logs", t, func() {
		stats, accessor, pub, pool, closeConn := buildContext(t)
		defer closeConn()

		done, cancel := runner.Run(stats)

		postfixutil.ReadFromTestReader(strings.NewReader(`
Jan 09 17:41:40 mail postfix/smtpd[1234]: disconnect from unknown[66.66.66.66] ehlo=1 auth=1 mail=1 rcpt=1 data=1 rset=1 commands=6
Jan 10 17:41:40 mail postfix/smtpd[1234]: disconnect from unknown[4.3.2.1] ehlo=1 auth=1 mail=1 rcpt=1 data=1 rset=1 commands=6
Jul 13 17:41:40 mail postfix/smtpd[26098]: disconnect from unknown[11.22.33.44] ehlo=1 auth=8/14 mail=1 rcpt=0/1 data=0/1 rset=1 commands=3/19
Sep  3 10:40:57 mail postfix/smtpd[8377]: disconnect from lalala.com[1002:1712:4e2b:d061:5dff:19f:c85f:a48f] ehlo=1 auth=0/1 commands=1/2
Sep  4 10:40:57 mail postfix/smtpd[9715]: disconnect from example.com[22.33.44.55] ehlo=1 auth=1 mail=1 rcpt=1 data=1 quit=1 commands=6
Dec 30 10:40:57 mail postfix/smtpd[4567]: disconnect from example.com[1.2.3.4] ehlo=1 auth=1 mail=1 rcpt=1 data=1 quit=1 commands=6
		`), pub, 2020, &timeutil.FakeClock{Time: timeutil.MustParseTime(`2020-12-31 00:00:00 +0000`)})

		// removes 2 entry older than 5months
		stats.Actions <- makeCleanAction(time.Hour*24*30*5, 2)

		// remove the remaining ones
		stats.Actions <- makeCleanAction(time.Hour*24*30*5, 10)

		cancel()
		So(done(), ShouldBeNil)

		{
			// we should get only the three last entries
			attempts, err := accessor.FetchAuthAttempts(context.Background(), timeutil.TimeInterval{
				From: timeutil.MustParseTime(`2020-01-01 00:00:00 +0000`),
				To:   timeutil.MustParseTime(`2020-12-31 00:00:00 +0000`),
			})

			So(err, ShouldBeNil)

			So(attempts.IPs, ShouldResemble, []string{"1.2.3.4", "1002:1712:4e2b:d061:5dff:19f:c85f:a48f", "22.33.44.55"})
			So(attempts.Attempts, ShouldResemble, []AttemptDesc{
				{Time: timeutil.MustParseTime(`2020-09-03 10:40:57 +0000`).Unix(), IPIndex: 1, Status: "failed"},
				{Time: timeutil.MustParseTime(`2020-09-04 10:40:57 +0000`).Unix(), IPIndex: 2, Status: "ok"},
				{Time: timeutil.MustParseTime(`2020-12-30 10:40:57 +0000`).Unix(), IPIndex: 0, Status: "ok"},
			})
		}

		conn, release := pool.Acquire()
		defer release()

		var (
			connectionsCount int
			commandsCount    int
		)

		So(conn.QueryRow(`select count(*) from connections`).Scan(&connectionsCount), ShouldBeNil)
		So(conn.QueryRow(`select count(*) from commands`).Scan(&commandsCount), ShouldBeNil)

		So(commandsCount, ShouldEqual, 14)
		So(connectionsCount, ShouldEqual, 3)
	})
}
