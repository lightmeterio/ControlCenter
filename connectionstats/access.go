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
	Time    int64  `json:"time"`
	IPIndex int    `json:"index"`
	Status  string `json:"status"`
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
		//nolint:sqlclosecheck
		sql, err := conn.Prepare(`
select
	count(connections.id)
from
	connections join commands
		on commands.connection_id = connections.id
where
	connections.disconnection_ts between ? and ? and commands.cmd = ?`)

		if err != nil {
			return errorutil.Wrap(err)
		}

		conn.Stmts[countQuery] = sql

		//nolint:sqlclosecheck
		sql, err = conn.Prepare(`
select
	ip, disconnection_ts as ts, success, total
from
	connections join commands
		on commands.connection_id = connections.id
where
	ts between ? and ? and commands.cmd = ?
	-- and commands.success != commands.total -- returns only attempts that failed
order by
	ts`)
		if err != nil {
			return errorutil.Wrap(err)
		}

		conn.Stmts[retrieveQuery] = sql

		return nil
	}); err != nil {
		return nil, errorutil.Wrap(err)
	}

	return &Accessor{pool: pool}, nil
}

func (a *Accessor) FetchAuthAttempts(ctx context.Context, interval timeutil.TimeInterval) (AccessResult, error) {
	conn, release := a.pool.Acquire()

	defer release()

	var count int

	if err := conn.Stmts[countQuery].QueryRowContext(ctx, interval.From.Unix(), interval.To.Unix(), AuthCommand).Scan(&count); err != nil {
		return AccessResult{}, errorutil.Wrap(err)
	}

	rows, err := conn.Stmts[retrieveQuery].QueryContext(ctx, interval.From.Unix(), interval.To.Unix(), AuthCommand)
	if err != nil {
		return AccessResult{}, errorutil.Wrap(err)
	}

	defer rows.Close()

	type rawAttemptDesc struct {
		time    int64
		ip      string
		success int
		total   int
	}

	rawAttempts := make([]rawAttemptDesc, 0, count)

	ipsSet := map[string]struct{}{}

	for rows.Next() {
		var (
			ip      net.IP
			ts      int64
			success int
			total   int
		)

		if err := rows.Scan(&ip, &ts, &success, &total); err != nil {
			return AccessResult{}, errorutil.Wrap(err)
		}

		ipAsString := ip.String()

		rawAttempts = append(rawAttempts, rawAttemptDesc{time: ts, ip: ipAsString, success: success, total: total})

		ipsSet[ipAsString] = struct{}{}
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
		times = append(times, AttemptDesc{Time: d.time, IPIndex: index, Status: statusFromStats(d.success, d.total)})
	}

	return AccessResult{
		IPs:      ips,
		Attempts: times,
	}, nil
}

func statusFromStats(success, total int) string {
	if success == total && total == 1 {
		return "ok"
	}

	if success == 0 {
		return "failed"
	}

	// NOTE: failed a few times, but then succeeded on authenticating.
	// this might indicate a password being cracked!
	return "suspicious"
}
