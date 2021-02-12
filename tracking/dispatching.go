// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package tracking

import (
	"database/sql"
	"github.com/rs/zerolog/log"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
)

const resultInfosCapacity = 512

type resultInfos struct {
	size   uint
	values [resultInfosCapacity]int64
}

func tryToDispatchAndReset(resultInfos *resultInfos, resultsToNotify chan<- resultInfos) {
	if resultInfos.size > 0 {
		resultsToNotify <- *resultInfos
		resultInfos.size = 0
	}
}

func dispatchAllResults(tracker *Tracker, resultsToNotify chan<- resultInfos, tx *sql.Tx) error {
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
		id       int64
	)

	count := 0
	resultInfos := resultInfos{}

	for {
		if resultInfos.size == resultInfosCapacity {
			tryToDispatchAndReset(&resultInfos, resultsToNotify)
		}

		if !rows.Next() {
			break
		}

		count++

		err = rows.Scan(&id, &resultId)

		if err != nil {
			return errorutil.Wrap(err)
		}

		// Yes, deleting while iterating over the results... That's supported by SQLite
		stmt := tx.Stmt(tracker.stmts[deleteFromNotificationQueues])

		defer func() {
			errorutil.MustSucceed(stmt.Close())
		}()

		_, err = stmt.Exec(id)
		if err != nil {
			return errorutil.Wrap(err)
		}

		// TODO: encapsulate it on add()
		resultInfos.values[resultInfos.size] = resultId
		resultInfos.size++
	}

	tryToDispatchAndReset(&resultInfos, resultsToNotify)

	if count > 0 {
		log.Info().Msgf("Dispatched a total of %v", count)
	}

	if err := rows.Err(); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}
