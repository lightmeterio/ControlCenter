// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package deliverydb

import (
	"database/sql"
	"errors"
	parser "gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser"
	"gitlab.com/lightmeter/controlcenter/tracking"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
)

func rowIdForQueue(queue string, tx *sql.Tx, stmts preparedStmts) (int64, error) {
	// maybe the queue already exists in the db
	stmt := tx.Stmt(stmts[findQueueByName])

	defer func() {
		errorutil.MustSucceed(stmt.Close())
	}()

	var queueId int64

	err := stmt.QueryRow(queue).Scan(&queueId)
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		// queue not found. Insert and get the rowid
		stmt := tx.Stmt(stmts[insertQueue])

		defer func() {
			errorutil.MustSucceed(stmt.Close())
		}()

		result, err := stmt.Exec(queue)
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

// TODO: a queue for a returned message should link to the original queue
func handleQueueInfo(deliveryRowId int64, tr tracking.Result, tx *sql.Tx, stmts preparedStmts) error {
	queue := tr[tracking.QueueDeliveryNameKey].Text()

	queueRowId, err := rowIdForQueue(queue, tx, stmts)
	if err != nil {
		return errorutil.Wrap(err)
	}

	// link delivery attempt to queue
	stmt := tx.Stmt(stmts[insertQueueDeliveryAttempt])

	defer func() {
		errorutil.MustSucceed(stmt.Close())
	}()

	if _, err := stmt.Exec(queueRowId, deliveryRowId); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func updateDeliveryStatusToExpired(queue string, tx *sql.Tx, stmts preparedStmts) error {
	stmt := tx.Stmt(stmts[updateDeliveryStatusByQueueName])

	defer func() {
		errorutil.MustSucceed(stmt.Close())
	}()

	if _, err := stmt.Exec(queue, parser.ExpiredStatus); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}