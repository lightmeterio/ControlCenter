// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package rawlogsdb

import (
	"database/sql"
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
	deleteOldLogEntriesKey

	lastStmtKey
)

var stmtsText = dbconn.StmtsText{
	insertLogLineKey: `insert into logs(time, checksum, content) values(?, ?, ?)`,

	// TODO: implement it!!! It gets two arguments: duration in seconds and a batch size
	deleteOldLogEntriesKey: `delete from logs where 1 = 2 and ? = ?`,
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
		//nolint:sqlclosecheck
		result, err := stmts.Get(deleteOldLogEntriesKey).Exec(maxAge/time.Second, batchSize)
		if err != nil {
			return errorutil.Wrap(err)
		}

		n, err := result.RowsAffected()
		if err != nil {
			return errorutil.Wrap(err)
		}

		if n > 0 {
			log.Debug().Msgf("Deleted %v raw log line entries", n)
		}

		return nil
	}
}
