// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package tracking

import (
	"database/sql"
	"github.com/rs/zerolog/log"
	"gitlab.com/lightmeter/controlcenter/data"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
)

func dispatchAllResults(tracker *Tracker, resultsToNotify chan<- resultInfo, tx *sql.Tx) error {
	stmt := tx.Stmt(tracker.stmts[selectFromNotificationQueues])

	defer func() {
		errorutil.MustSucceed(stmt.Close())
	}()

	// NOTE: as usual, the rowserrcheck is not able to see rows.Err() is called below :-(
	//nolint:rowserrcheck
	rows, err := stmt.Query()

	if err != nil {
		return errorutil.Wrap(err)
	}

	defer func() {
		errorutil.MustSucceed(rows.Close())
	}()

	var (
		resultId int64
		line     uint64 // NOTE: it's stored as an int64 in SQLite... :-(
		filename string
		id       int64
	)

	count := 0

	for rows.Next() {
		count++

		err = rows.Scan(&id, &resultId, &filename, &line)

		if err != nil {
			return errorutil.Wrap(err)
		}

		resultInfo := resultInfo{id: resultId, loc: data.RecordLocation{Line: line, Filename: filename}}

		resultsToNotify <- resultInfo

		// Yes, deleting while iterating over the results... That's supported by SQLite
		stmt := tx.Stmt(tracker.stmts[deleteFromNotificationQueues])

		defer func() {
			errorutil.MustSucceed(stmt.Close())
		}()

		_, err = stmt.Exec(id)
		if err != nil {
			return errorutil.Wrap(err)
		}
	}

	if err := rows.Err(); err != nil {
		return errorutil.Wrap(err)
	}

	if count > 0 {
		log.Debug().Msgf("Tracker has dispatched %v messages", count)
	}

	return nil
}
