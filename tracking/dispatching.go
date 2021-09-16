// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package tracking

import (
	"github.com/rs/zerolog/log"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"time"
)

// NOTE: this is an arbitrary value, found during manual testing,
// and was the one with the best performance
const resultInfosCapacity = 4

type resultInfos struct {
	batchId int64
	id      int64
	size    uint
	values  [resultInfosCapacity]int64
}

func tryToDispatchAndReset(resultInfos *resultInfos, resultsToNotify chan<- resultInfos) {
	if resultInfos.size > 0 {
		resultsToNotify <- *resultInfos
		resultInfos.size = 0
		resultInfos.id++
	}
}

func dispatchAllResults(resultsToNotify chan<- resultInfos, batchId int64, trackerStmts dbconn.TxPreparedStmts) error {
	start := time.Now()

	// NOTE: as usual, the rowserrcheck is not able to see rows.Err() is called below :-(
	//nolint:rowserrcheck
	rows, err := trackerStmts.Get(selectFromNotificationQueues).Query()

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
	resultInfos := resultInfos{batchId: batchId}

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
		_, err = trackerStmts.Get(deleteFromNotificationQueues).Exec(id)
		if err != nil {
			return errorutil.Wrap(err)
		}

		// TODO: encapsulate it on add()
		resultInfos.values[resultInfos.size] = resultId
		resultInfos.size++
	}

	tryToDispatchAndReset(&resultInfos, resultsToNotify)

	if count > 0 {
		log.Debug().Msgf("Dispatched a total of %v on batch %v in %v", count, batchId, time.Since(start))
	}

	if err := rows.Err(); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}
