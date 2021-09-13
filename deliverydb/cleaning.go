// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package deliverydb

import (
	"database/sql"
	"gitlab.com/lightmeter/controlcenter/pkg/dbrunner"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"time"
)

func tryToDeleteMessageId(tx *sql.Tx, messageId int64, stmts dbrunner.PreparedStmts) error {
	var msgIdsCount int

	stmt := tx.Stmt(stmts[countDeliveriesWithMessageId])
	defer stmt.Close()

	// is it the only delivery with this message-id?
	if err := stmt.QueryRow(messageId).Scan(&msgIdsCount); err != nil {
		return errorutil.Wrap(err)
	}

	if msgIdsCount > 1 {
		// do not delete messageid, as there are more messages using it
		return nil
	}

	stmt = tx.Stmt(stmts[deleteMessageIdById])
	defer stmt.Close()

	if _, err := stmt.Exec(messageId); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func tryToDeleteDeliveryQueue(tx *sql.Tx, deliveryId int64, stmts dbrunner.PreparedStmts) error {
	var (
		queueId                 int64
		deliveryQueueRelationId int64
	)

	stmt := tx.Stmt(stmts[selectQueueIdForDeliveryId])
	defer stmt.Close()

	if err := stmt.QueryRow(deliveryId).Scan(&deliveryQueueRelationId, &queueId); err != nil {
		return errorutil.Wrap(err)
	}

	var queueCount int

	// is it the only delivery in a given queue?
	stmt = tx.Stmt(stmts[countDeliveriesWithQueue])
	defer stmt.Close()

	if err := stmt.QueryRow(queueId).Scan(&queueCount); err != nil {
		return errorutil.Wrap(err)
	}

	stmt = tx.Stmt(stmts[deleteDeliveryQueueById])
	defer stmt.Close()

	// Remove the delivery X queue relationship
	if _, err := stmt.Exec(deliveryQueueRelationId); err != nil {
		return errorutil.Wrap(err)
	}

	if queueCount > 1 {
		// do not delete queue, as there are more messages on it
		return nil
	}

	// from here on, should delete the queue.

	stmt = tx.Stmt(stmts[deleteExpiredQueuesByQueueId])
	defer stmt.Close()

	// delete the expired info entry, if it exists
	if _, err := stmt.Exec(queueId); err != nil {
		return errorutil.Wrap(err)
	}

	stmt = tx.Stmt(stmts[deleteQueueParentingByQueueId])
	defer stmt.Close()

	// NOTE: this is a risky move, as some "dangling" relationships might be create as result for some time,
	// although they are very unlikely and will always eventually be removed on the next "cleanup" call
	if _, err := stmt.Exec(queueId, queueId); err != nil {
		return errorutil.Wrap(err)
	}

	stmt = tx.Stmt(stmts[deleteQueueById])
	defer stmt.Close()

	// Finally, remove the queue
	if _, err := stmt.Exec(queueId); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func makeCleanAction(maxAge time.Duration) dbrunner.Action {
	return func(tx *sql.Tx, stmts dbrunner.PreparedStmts) error {
		// NOTE: the time in the database is in Seconds
		stmt := tx.Stmt(stmts[selectOldDeliveries])
		defer stmt.Close()

		rows, err := stmt.Query(maxAge / time.Second)
		if err != nil {
			return errorutil.Wrap(err)
		}

		defer rows.Close()

		for rows.Next() {
			var (
				deliveryId int64
				messageId  int64
			)

			if err := rows.Scan(&deliveryId, &messageId); err != nil {
				return errorutil.Wrap(err)
			}

			if err := tryToDeleteMessageId(tx, messageId, stmts); err != nil {
				return errorutil.Wrap(err)
			}

			if err := tryToDeleteDeliveryQueue(tx, deliveryId, stmts); err != nil {
				return errorutil.Wrap(err)
			}

			// TODO:
			// maybe delete sender and recipient and orig_recipient domain parts,
			// although they are very unlikely to grow over time

			stmt := tx.Stmt(stmts[deleteOldDeliveries])
			defer stmt.Close()

			if _, err := stmt.Exec(deliveryId); err != nil {
				return errorutil.Wrap(err)
			}
		}

		if err := rows.Err(); err != nil {
			return errorutil.Wrap(err)
		}

		return nil
	}
}
