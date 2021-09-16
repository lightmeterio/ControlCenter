// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package deliverydb

import (
	"database/sql"
	"errors"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/tracking"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
)

func rowIdForQueue(queue string, tx *sql.Tx, stmts dbconn.TxPreparedStmts) (int64, error) {
	var queueId int64

	err := stmts.Get(findQueueByName).QueryRow(queue).Scan(&queueId)
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		result, err := stmts.Get(insertQueue).Exec(queue)
		if err != nil {
			return 0, errorutil.Wrap(err)
		}

		rowId, err := result.LastInsertId()
		if err != nil {
			return 0, errorutil.Wrap(err)
		}

		return rowId, nil
	}

	if err != nil {
		return 0, errorutil.Wrap(err)
	}

	return queueId, nil
}

func handleQueueInfo(deliveryRowId int64, tr tracking.Result, tx *sql.Tx, stmts dbconn.TxPreparedStmts) error {
	queue := tr[tracking.QueueDeliveryNameKey].Text()

	queueRowId, err := rowIdForQueue(queue, tx, stmts)
	if err != nil {
		return errorutil.Wrap(err)
	}

	if _, err := stmts.Get(insertQueueDeliveryAttempt).Exec(queueRowId, deliveryRowId); err != nil {
		return errorutil.Wrap(err)
	}

	parentQueue := tr[tracking.ParentQueueDeliveryNameKey]

	if parentQueue.IsNone() {
		return nil
	}

	parentQueueId, err := rowIdForQueue(parentQueue.Text(), tx, stmts)
	if err != nil {
		return errorutil.Wrap(err)
	}

	if _, err := stmts.Get(insertQueueParenting).Exec(parentQueueId, queueRowId, QueueParentingTypeReturnedToSender); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

type QueueParentingType int

const (
	// NOTE: this value is stored in the database, so never change it unless you want to break backward compatibility!
	QueueParentingTypeReturnedToSender QueueParentingType = 1
)

func setQueueExpired(queue string, expiredTs int64, tx *sql.Tx, stmts dbconn.TxPreparedStmts) error {
	queueId, err := rowIdForQueue(queue, tx, stmts)
	if err != nil {
		return errorutil.Wrap(err)
	}

	if _, err := stmts.Get(insertExpiredQueue).Exec(queueId, expiredTs); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}
