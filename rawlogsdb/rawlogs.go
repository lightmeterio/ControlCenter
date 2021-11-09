// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package rawlogsdb

import (
	"database/sql"
	"errors"
	"github.com/rs/zerolog/log"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/pkg/dbrunner"
	"gitlab.com/lightmeter/controlcenter/pkg/postfix"
	_ "gitlab.com/lightmeter/controlcenter/rawlogsdb/migrations"
	"gitlab.com/lightmeter/controlcenter/util/closeutil"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"time"
)

type DB struct {
	dbrunner.Runner
	closeutil.Closers
}

const (
	insertLogLineKey = iota
	selectMostRecentLogTimeKey
	selectOldestLogEntriesKey
	deleteLogEntryKey
	lastStmtKey
)

var stmtsText = dbconn.StmtsText{
	insertLogLineKey:           `insert into logs(time, checksum, content) values(?, ?, ?)`,
	selectMostRecentLogTimeKey: `select time from logs order by time desc limit 1`,
	selectOldestLogEntriesKey:  `select id from logs where time < ? order by time, id asc limit ?`,
	deleteLogEntryKey:          `delete from logs where id = ?`,
}

func New(conn dbconn.RwConn) (*DB, error) {
	stmts := dbconn.BuildPreparedStmts(lastStmtKey)

	if err := dbconn.PrepareRwStmts(stmtsText, conn, &stmts); err != nil {
		return nil, errorutil.Wrap(err)
	}

	const (
		// ~3 months. TODO: make it configurable
		maxAge            = (time.Hour * 24 * 30 * 3)
		cleaningBatchSize = 10000
		cleaningFrequency = time.Second * 30
	)

	return &DB{
		Runner:  dbrunner.New(500*time.Millisecond, 1024*1000, conn, stmts, cleaningFrequency, makeCleanAction(maxAge, cleaningBatchSize)),
		Closers: closeutil.New(stmts),
	}, nil
}

func (db *DB) Publisher() postfix.Publisher {
	return &publisher{actions: db.Actions}
}

type publisher struct {
	actions chan<- dbrunner.Action
}

func (pub *publisher) Publish(r postfix.Record) {
	pub.actions <- func(tx *sql.Tx, stmts dbconn.TxPreparedStmts) (err error) {
		//nolint:sqlclosecheck
		if _, err := stmts.Get(insertLogLineKey).Exec(r.Time.Unix(), r.Sum, r.Line); err != nil {
			return errorutil.Wrap(err)
		}

		return nil
	}
}

func makeCleanAction(maxAge time.Duration, batchSize int) dbrunner.Action {
	return func(tx *sql.Tx, stmts dbconn.TxPreparedStmts) error {
		var mostRecentLogTime int64

		// FIXME: this process is way too complicated and should be simplified.
		// To sum up, what it does is: delete maximum of batchSize of the oldest log entries
		// which are older than the time of the most recent log entry subtracted maxAge.
		// I could not come up with an efficient query for it, so had to break it into "small pieces",
		// doing some of the computation in Go instead of SQL.
		// I imagine that such query would look like the pseudo-sql:
		// delete at most <batchSize> from logs where time < ((select max(time) from logs) - <maxAge>)

		//nolint:sqlclosecheck
		err := stmts.Get(selectMostRecentLogTimeKey).QueryRow().Scan(&mostRecentLogTime)

		// do nothing if there are no lines to be cleaned
		if err != nil && errors.Is(err, sql.ErrNoRows) {
			return nil
		}

		if err != nil {
			return errorutil.Wrap(err)
		}

		oldestTimeToKeep := time.Unix(mostRecentLogTime, 0).Add(-maxAge).Unix()

		//nolint:sqlclosecheck
		rows, err := stmts.Get(selectOldestLogEntriesKey).Query(oldestTimeToKeep, batchSize)
		if err != nil {
			return errorutil.Wrap(err)
		}

		defer rows.Close()

		n := 0

		for rows.Next() {
			var id int64

			if err := rows.Scan(&id); err != nil {
				return errorutil.Wrap(err)
			}

			//nolint:sqlclosecheck
			if _, err := stmts.Get(deleteLogEntryKey).Exec(id); err != nil {
				return errorutil.Wrap(err)
			}

			n++
		}

		if err := rows.Err(); err != nil {
			return errorutil.Wrap(err)
		}

		if n > 0 {
			log.Debug().Msgf("Deleted %v raw log line entries", n)
		}

		return nil
	}
}
