// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package deliverydb

import (
	"database/sql"
	"errors"
	"github.com/rs/zerolog/log"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/pkg/dbrunner"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"time"
)

func tryToDeleteMessageId(tx *sql.Tx, messageId int64, deliveryTime int64, stmts dbconn.TxPreparedStmts) error {
	var msgIdsCount int

	// In order not to scan the whole messageids table for messages with the same id,
	// we scan only a small window of +- 2 days, as results are very unlikely outside of
	// a short window.
	// TODO: For the vast majority of cases, there is only one message-id for each message
	// meaning that we should make message-id a field of the deliveries table instead of an external
	// table, to avoid this check here...
	// For the few messages with same message-ids, the duplication is still worth it.
	toleranceTimeForSameMessageIdInSeconds := (time.Hour * 24 * 2) / time.Second

	maxTime := deliveryTime + int64(toleranceTimeForSameMessageIdInSeconds)

	// is it the only delivery with this message-id?
	//nolint:sqlclosecheck
	if err := stmts.Get(countDeliveriesWithMessageId).QueryRow(messageId, maxTime).Scan(&msgIdsCount); err != nil {
		return errorutil.Wrap(err)
	}

	if msgIdsCount > 1 {
		// do not delete messageid, as there are more messages using it
		return nil
	}

	//nolint:sqlclosecheck
	if _, err := stmts.Get(deleteMessageIdById).Exec(messageId); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func tryToDeleteDeliveryQueue(tx *sql.Tx, deliveryId int64, stmts dbconn.TxPreparedStmts) error {
	var (
		queueId                 int64
		deliveryQueueRelationId int64
	)

	//nolint:sqlclosecheck
	err := stmts.Get(selectQueueIdForDeliveryId).QueryRow(deliveryId).Scan(&deliveryQueueRelationId, &queueId)

	// NOTE: for entries before we implemented storing the queue info in the database (a07e3ce76e43bfbb6b5a2a5ef5508b93234bed61),
	// this query will return an empty result, so we just skip them.
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		return nil
	}

	if err != nil {
		return errorutil.Wrap(err)
	}

	var queueCount int

	//nolint:sqlclosecheck
	if err := stmts.Get(countDeliveriesWithQueue).QueryRow(queueId).Scan(&queueCount); err != nil {
		return errorutil.Wrap(err)
	}

	// Remove the delivery X queue relationship
	//nolint:sqlclosecheck
	if _, err := stmts.Get(deleteDeliveryQueueById).Exec(deliveryQueueRelationId); err != nil {
		return errorutil.Wrap(err)
	}

	if queueCount > 1 {
		// do not delete queue, as there are more messages on it
		return nil
	}

	// from here on, should delete the queue.

	// delete the expired info entry, if it exists
	//nolint:sqlclosecheck
	if _, err := stmts.Get(deleteExpiredQueuesByQueueId).Exec(queueId); err != nil {
		return errorutil.Wrap(err)
	}

	// NOTE: this is a risky move, as some "dangling" relationships might be create as result for some time,
	// although they are very unlikely and will always eventually be removed on the next "cleanup" call
	//nolint:sqlclosecheck
	if _, err := stmts.Get(deleteQueueParentingByQueueId).Exec(queueId, queueId); err != nil {
		return errorutil.Wrap(err)
	}

	// Finally, remove the queue
	//nolint:sqlclosecheck
	if _, err := stmts.Get(deleteQueueById).Exec(queueId); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func makeCleanAction(maxAge time.Duration, batchSize int) dbrunner.Action {
	return func(tx *sql.Tx, stmts dbconn.TxPreparedStmts) (err error) {
		// NOTE: the time in the database is in Seconds
		//nolint:sqlclosecheck
		rows, err := stmts.Get(selectOldDeliveries).Query(maxAge/time.Second, batchSize)
		if err != nil {
			return errorutil.Wrap(err)
		}

		defer errorutil.UpdateErrorFromCloser(rows, &err)

		timeBeforeAction := time.Now()

		numberOfDeletedRecords := 0

		for rows.Next() {
			numberOfDeletedRecords++

			var (
				deliveryId   int64
				messageId    int64
				deliveryTime int64
			)

			if err := rows.Scan(&deliveryId, &messageId, &deliveryTime); err != nil {
				return errorutil.Wrap(err)
			}

			if err := tryToDeleteMessageId(tx, messageId, deliveryTime, stmts); err != nil {
				return errorutil.Wrap(err)
			}

			if err := tryToDeleteDeliveryQueue(tx, deliveryId, stmts); err != nil {
				return errorutil.Wrap(err)
			}

			// TODO:
			// maybe delete sender and recipient and orig_recipient domain parts,
			// although they are very unlikely to grow too much over time

			//nolint:sqlclosecheck
			if _, err := stmts.Get(deleteOldDeliveries).Exec(deliveryId); err != nil {
				return errorutil.Wrap(err)
			}
		}

		if err := rows.Err(); err != nil {
			return errorutil.Wrap(err)
		}

		if numberOfDeletedRecords > 0 {
			log.Info().Msgf("Deleted %v records in %v", numberOfDeletedRecords, time.Since(timeBeforeAction))
		}

		return nil
	}
}
