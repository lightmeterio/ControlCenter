// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package connectionstats

import (
	"context"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"net"
	"sort"
)

type AttemptDesc struct {
	Time     int64    `json:"time"`
	IPIndex  int      `json:"index"`
	Status   string   `json:"status"`
	Protocol Protocol `json:"protocol"`
}

type AccessResult struct {
	IPs      []string      `json:"ips"`
	Attempts []AttemptDesc `json:"attempts"`
}

type Accessor struct {
	pool *dbconn.RoPool
}

const (
	countQuery = iota
	retrieveQuery
)

func NewAccessor(pool *dbconn.RoPool) (*Accessor, error) {
	if err := pool.ForEach(func(conn *dbconn.RoPooledConn) error {
		if err := conn.PrepareStmt(`
select
	count(connections.id)
from
	connections join commands
		on commands.connection_id = connections.id
where
	connections.disconnection_ts between ? and ? and commands.cmd = ?`, countQuery); err != nil {
			return errorutil.Wrap(err)
		}

		if err := conn.PrepareStmt(`
select
	ip, cmd, disconnection_ts as ts, success, total, protocol
from
	connections join commands
		on commands.connection_id = connections.id
where
	ts between ? and ? and commands.cmd in (?, ?, ?)
order by
	ts`, retrieveQuery); err != nil {
			return errorutil.Wrap(err)
		}

		return nil
	}); err != nil {
		return nil, errorutil.Wrap(err)
	}

	return &Accessor{pool: pool}, nil
}

func (a *Accessor) FetchAuthAttempts(ctx context.Context, interval timeutil.TimeInterval) (result AccessResult, err error) {
	conn, release, err := a.pool.AcquireContext(ctx)
	if err != nil {
		return AccessResult{}, errorutil.Wrap(err)
	}

	defer release()

	var count int

	//nolint:sqlclosecheck
	if err := conn.GetStmt(countQuery).QueryRowContext(ctx, interval.From.Unix(), interval.To.Unix(), AuthCommand).Scan(&count); err != nil {
		return AccessResult{}, errorutil.Wrap(err)
	}

	//nolint:sqlclosecheck
	rows, err := conn.GetStmt(retrieveQuery).QueryContext(ctx, interval.From.Unix(), interval.To.Unix(), AuthCommand, DovecotAuthCommand, DovecotBlockCommand)
	if err != nil {
		return AccessResult{}, errorutil.Wrap(err)
	}

	defer errorutil.UpdateErrorFromCloser(rows, &err)

	type rawAttemptDesc struct {
		time     int64
		command  Command
		ip       string
		success  int
		total    int
		protocol Protocol
	}

	rawAttempts := make([]rawAttemptDesc, 0, count)

	ipsSet := map[string]struct{}{}

	for rows.Next() {
		var (
			ip       net.IP
			command  Command
			ts       int64
			success  int
			total    int
			protocol Protocol
		)

		if err := rows.Scan(&ip, &command, &ts, &success, &total, &protocol); err != nil {
			return AccessResult{}, errorutil.Wrap(err)
		}

		ipAsString := ip.String()

		rawAttempts = append(rawAttempts, rawAttemptDesc{
			time:     ts,
			ip:       ipAsString,
			command:  command,
			success:  success,
			total:    total,
			protocol: protocol,
		})

		ipsSet[ipAsString] = struct{}{}
	}

	if err := rows.Err(); err != nil {
		return AccessResult{}, errorutil.Wrap(err)
	}

	ips := make([]string, 0, len(ipsSet))

	for ip := range ipsSet {
		ips = append(ips, ip)
	}

	// sets are not guaranteed to be ordered, so we make some order!
	sort.Strings(ips)

	ipIndexes := make(map[string]int, len(ips))

	for i, v := range ips {
		ipIndexes[v] = i
	}

	times := make([]AttemptDesc, 0, len(rawAttempts))

	for _, d := range rawAttempts {
		index := ipIndexes[d.ip]
		times = append(times, AttemptDesc{
			Time:     d.time,
			IPIndex:  index,
			Status:   statusFromStats(d.command, d.success, d.total),
			Protocol: d.protocol,
		})
	}

	return AccessResult{
		IPs:      ips,
		Attempts: times,
	}, nil
}

func statusFromStats(command Command, success, total int) string {
	if success == total && total == 1 {
		return "ok"
	}

	if success != 0 {
		// NOTE: failed a few times, but then succeeded on authenticating.
		// this might indicate a password being cracked!
		return "suspicious"
	}

	if command == DovecotBlockCommand {
		return "blocked"
	}

	return "failed"
}
