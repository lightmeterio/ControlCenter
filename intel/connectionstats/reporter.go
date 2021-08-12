// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package connectionstats

import (
	"database/sql"
	"gitlab.com/lightmeter/controlcenter/connectionstats"
	"gitlab.com/lightmeter/controlcenter/intel/collector"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"time"
)

type stats struct {
	Success int `json:"success"`
	Total   int `json:"total"`
}

type entry struct {
	usedOnAuth bool
	Time       time.Time                         `json:"time"`
	IP         string                            `json:"ip"`
	Commands   map[connectionstats.Command]stats `json:"commands"`
}

type report struct {
	Interval timeutil.TimeInterval `json:"time_interval"`
	Entries  []entry               `json:"entries"`
}

type Reporter struct {
	pool *dbconn.RoPool
}

// New receives a connection to a `connectionstats` database.
func NewReporter(pool *dbconn.RoPool) *Reporter {
	return &Reporter{pool: pool}
}

const executionInterval = 10 * time.Minute

func (r *Reporter) ExecutionInterval() time.Duration {
	return executionInterval
}

func (r *Reporter) Close() error {
	return nil
}

func (r *Reporter) Step(tx *sql.Tx, clock timeutil.Clock) error {
	conn, release := r.pool.Acquire()

	defer release()

	interval := timeutil.TimeInterval{From: clock.Now().Add(-executionInterval), To: clock.Now()}

	connectionsStmt, err := conn.Prepare(`select id, lm_ip_to_string(ip), disconnection_ts from connections where disconnection_ts between ? and ?`)
	if err != nil {
		return errorutil.Wrap(err)
	}

	defer connectionsStmt.Close()

	commandsStmt, err := conn.Prepare(`select cmd, success, total from commands where connection_id = ?`)
	if err != nil {
		return errorutil.Wrap(err)
	}

	defer commandsStmt.Close()

	entries := []entry{}

	results, err := connectionsStmt.Query(interval.From.Unix(), interval.To.Unix())
	if err != nil {
		return errorutil.Wrap(err)
	}

	defer results.Close()

	for results.Next() {
		entry, err := entryForQueryResult(results, commandsStmt)
		if err != nil {
			return errorutil.Wrap(err)
		}

		if entry.usedOnAuth {
			entries = append(entries, entry)
		}
	}

	if err := results.Err(); err != nil {
		return errorutil.Wrap(err)
	}

	report := report{
		Interval: interval,
		Entries:  entries,
	}

	if err := collector.Collect(tx, clock, r.ID(), &report); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func entryForQueryResult(results *sql.Rows, commandsStmt *sql.Stmt) (entry, error) {
	var (
		id int64
		ts int64
		ip string
	)

	if err := results.Scan(&id, &ip, &ts); err != nil {
		return entry{}, errorutil.Wrap(err)
	}

	commandsResults, err := commandsStmt.Query(id)
	if err != nil {
		return entry{}, errorutil.Wrap(err)
	}

	defer commandsResults.Close()

	e := entry{
		Time:     time.Unix(ts, 0).In(time.UTC),
		IP:       ip,
		Commands: map[connectionstats.Command]stats{},
	}

	for commandsResults.Next() {
		var (
			cmd     int
			success int
			total   int
		)

		if err := commandsResults.Scan(&cmd, &success, &total); err != nil {
			return entry{}, errorutil.Wrap(err)
		}

		e.Commands[connectionstats.Command(cmd)] = stats{
			Success: success,
			Total:   total,
		}
	}

	// we are interested only on connections that tried (even if failed) to authenticate
	e.usedOnAuth = func() bool {
		for k := range e.Commands {
			if k == connectionstats.AuthCommand {
				return true
			}
		}

		return false
	}()

	if err := commandsResults.Err(); err != nil {
		return entry{}, errorutil.Wrap(err)
	}

	return e, nil
}

func (r *Reporter) ID() string {
	return "connection_stats_with_auth"
}
