// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package core

import (
	_ "gitlab.com/lightmeter/controlcenter/intel/migrations"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/pkg/dbrunner"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"

	"database/sql"
	"errors"
	"time"
)

type Options struct {
	// How often should the c
	CycleInterval time.Duration

	// How often should the reports be dispatched/sent?
	ReportInterval time.Duration

	RetentionDuration time.Duration
}

func NewRunner(conn dbconn.RwConn, options Options) *dbrunner.Runner {
	stmts := dbconn.PreparedStmts{}
	return dbrunner.New(options.CycleInterval, 10, conn, stmts, time.Hour*12, MakeCleanAction(options.RetentionDuration))
}

func MakeCleanAction(maxAge time.Duration) dbrunner.Action {
	actions := []dbrunner.Action{func(tx *sql.Tx, stmts dbconn.TxPreparedStmts) error {
		var mostRecentDispatchTime int64
		err := tx.QueryRow(`select time from queued_reports order by id desc limit 1`).Scan(&mostRecentDispatchTime)

		if err != nil && errors.Is(err, sql.ErrNoRows) {
			return nil
		}

		if err != nil {
			return errorutil.Wrap(err)
		}

		mostRecentTime := time.Unix(mostRecentDispatchTime, 0)
		oldestTimeToKeep := mostRecentTime.Add(-maxAge)
		oldestTimeToKeepInTimestamp := oldestTimeToKeep.Unix()

		if _, err := tx.Exec(`delete from queued_reports where time < ?`, oldestTimeToKeepInTimestamp); err != nil {
			return errorutil.Wrap(err)
		}

		if _, err := tx.Exec(`delete from dispatch_times where time < ?`, oldestTimeToKeepInTimestamp); err != nil {
			return errorutil.Wrap(err)
		}

		return nil
	}, func(tx *sql.Tx, stmts dbconn.TxPreparedStmts) error {
		const query = `with time_cut as (
			select
				(received_time - ?) as v
			from
				events
			order by
				id desc
			limit 1
		)
		delete from events where received_time < (select v from time_cut)`

		if _, err := tx.Exec(query, maxAge/time.Second); err != nil {
			return errorutil.Wrap(err)
		}

		return nil
	}}

	return func(tx *sql.Tx, stmts dbconn.TxPreparedStmts) error {
		for _, action := range actions {
			if err := action(tx, stmts); err != nil {
				return errorutil.Wrap(err)
			}
		}

		return nil
	}
}
