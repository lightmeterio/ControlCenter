// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package tracking

import (
	"database/sql"
	"errors"
	"gitlab.com/lightmeter/controlcenter/pkg/postfix"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
)

func collectKeyValueResult(result *Result, stmt *sql.Stmt, args ...interface{}) error {
	var (
		id    int64
		key   int
		value interface{}
	)

	//nolint:rowserrcheck
	rows, err := stmt.Query(args...)
	if err != nil {
		return errorutil.Wrap(err)
	}

	defer func() {
		errorutil.MustSucceed(rows.Close())
	}()

	for rows.Next() {
		err = rows.Scan(&id, &key, &value)
		if err != nil {
			return errorutil.Wrap(err)
		}

		// TODO: abort if the key is not a valid result key (out of index)
		result[key] = ResultEntryFromValue(value)
	}

	if err := rows.Err(); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

type DeletionError struct {
	Err *errorutil.Error
	Loc postfix.RecordLocation
}

func (e *DeletionError) Unwrap() error {
	return e.Err
}

func (e *DeletionError) Error() string {
	return e.Err.Error()
}

func tryToDeleteQueue(tx *sql.Tx, trackerStmts trackerStmts, queueId int64, loc postfix.RecordLocation) (bool, error) {
	deleted, err := tryToDeleteQueueNotIgnoringErrors(tx, trackerStmts, queueId, loc)

	// Treat deletion errors (some queries return "norows") differently for now...
	if err != nil && (errors.Is(err, sql.ErrNoRows) || errors.Is(err, ErrInvalidAffectedLines)) {
		return false, &DeletionError{Err: errorutil.Wrap(err), Loc: loc}
	}

	if err != nil {
		return false, errorutil.Wrap(err)
	}

	return deleted, nil
}

func tryToDeleteQueueNotIgnoringErrors(tx *sql.Tx, trackerStmts trackerStmts, queueId int64, loc postfix.RecordLocation) (bool, error) {
	err := decrementQueueUsage(tx, trackerStmts, queueId)
	if err != nil {
		return false, errorutil.Wrap(err)
	}

	var usageCounter int

	stmt := tx.Stmt(trackerStmts[queueUsageCounter])

	defer func() {
		errorutil.MustSucceed(stmt.Close())
	}()

	err = stmt.QueryRow(queueId).Scan(&usageCounter)
	if err != nil {
		return false, errorutil.Wrap(err)
	}

	if usageCounter > 0 {
		return false, nil
	}

	// Check if there's any queue that depends on me.
	// In such scenario, I cannot be deleted yet
	var countDependentQueues int64

	stmt = tx.Stmt(trackerStmts[countNewQueueFromParenting])

	defer func() {
		errorutil.MustSucceed(stmt.Close())
	}()

	err = stmt.QueryRow(queueId).Scan(&countDependentQueues)
	if err != nil {
		return false, errorutil.Wrap(err)
	}

	if countDependentQueues > 0 {
		return false, nil
	}

	err = deleteQueueRec(tx, trackerStmts, queueId)
	if err != nil {
		return false, errorutil.Wrap(err)
	}

	return true, nil
}

func deleteQueueRec(tx *sql.Tx, trackerStmts trackerStmts, queueId int64) error {
	stmt := tx.Stmt(trackerStmts[selectQueueFromParentingNewQueue])

	defer func() {
		errorutil.MustSucceed(stmt.Close())
	}()

	//nolint:rowserrcheck
	rows, err := stmt.Query(queueId)
	if err != nil {
		return errorutil.Wrap(err)
	}

	defer func() {
		errorutil.MustSucceed(rows.Close())
	}()

	var (
		dependencyQueueId int64
		id                int64
	)

	for rows.Next() {
		err = rows.Scan(&id, &dependencyQueueId)
		if err != nil {
			return errorutil.Wrap(err)
		}

		err = deleteQueueRec(tx, trackerStmts, dependencyQueueId)
		if err != nil {
			return errorutil.Wrap(err)
		}

		stmt := tx.Stmt(trackerStmts[deleteQueueParentingById])

		defer func() {
			errorutil.MustSucceed(stmt.Close())
		}()

		_, err = stmt.Exec(id)
		if err != nil {
			return errorutil.Wrap(err)
		}
	}

	err = rows.Err()
	if err != nil {
		return errorutil.Wrap(err)
	}

	err = tryToDeleteConnectionForQueue(tx, trackerStmts, queueId)
	if err != nil {
		return errorutil.Wrap(err)
	}

	err = deleteQueue(tx, trackerStmts, queueId)
	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func countQueuesOnConnection(tx *sql.Tx, trackerStmts trackerStmts, connectionId int64) (int, error) {
	var connectionCounter int

	stmt := tx.Stmt(trackerStmts[countConnectionUsageById])

	defer func() {
		errorutil.MustSucceed(stmt.Close())
	}()

	err := stmt.QueryRow(connectionId).Scan(&connectionCounter)
	if err != nil {
		return 0, errorutil.Wrap(err)
	}

	return connectionCounter, nil
}

func tryToDeleteConnectionForQueue(tx *sql.Tx, trackerStmts trackerStmts, queueId int64) error {
	var connectionId int64

	stmt := tx.Stmt(trackerStmts[connectionIdForQueue])

	defer func() {
		errorutil.MustSucceed(stmt.Close())
	}()

	err := stmt.QueryRow(queueId).Scan(&connectionId)
	if err != nil {
		return errorutil.Wrap(err)
	}

	err = decrementConnectionUsage(tx, trackerStmts, connectionId)
	if err != nil {
		return errorutil.Wrap(err)
	}

	connectionCounter, err := countQueuesOnConnection(tx, trackerStmts, connectionId)
	if err != nil {
		return errorutil.Wrap(err)
	}

	if connectionCounter > 0 {
		return nil
	}

	// TODO: do not delete connection if it's still active (no disconnect command has been done for it)

	err = deleteConnection(tx, trackerStmts, connectionId)
	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func deleteConnection(tx *sql.Tx, trackerStmts trackerStmts, connectionId int64) error {
	err := tryToDeletePidForConnection(tx, trackerStmts, connectionId)
	if err != nil {
		return errorutil.Wrap(err)
	}

	stmt := tx.Stmt(trackerStmts[deleteConnectionDataByConnectionId])

	defer func() {
		errorutil.MustSucceed(stmt.Close())
	}()

	_, err = stmt.Exec(connectionId)
	if err != nil {
		return errorutil.Wrap(err)
	}

	stmt = tx.Stmt(trackerStmts[deleteConnectionById])

	defer func() {
		errorutil.MustSucceed(stmt.Close())
	}()

	_, err = stmt.Exec(connectionId)
	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func countPidUsage(tx *sql.Tx, trackerStmts trackerStmts, pidId int64) (int, error) {
	var count int

	stmt := tx.Stmt(trackerStmts[countPidUsageByPidId])

	defer func() {
		errorutil.MustSucceed(stmt.Close())
	}()

	err := stmt.QueryRow(pidId).Scan(&count)
	if err != nil {
		return 0, errorutil.Wrap(err)
	}

	return count, nil
}

func incrementPidUsage(tx *sql.Tx, stmts trackerStmts, pidId int64) error {
	stmt := tx.Stmt(stmts[incrementPidUsageById])

	defer func() {
		errorutil.MustSucceed(stmt.Close())
	}()

	_, err := stmt.Exec(pidId)
	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func decrementPidUsage(tx *sql.Tx, stmts trackerStmts, pidId int64) error {
	stmt := tx.Stmt(stmts[decrementPidUsageById])

	defer func() {
		errorutil.MustSucceed(stmt.Close())
	}()

	_, err := stmt.Exec(pidId)
	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func tryToDeletePidForConnection(tx *sql.Tx, trackerStmts trackerStmts, connectionId int64) error {
	var pidId int64

	stmt := tx.Stmt(trackerStmts[pidIdForConnection])

	defer func() {
		errorutil.MustSucceed(stmt.Close())
	}()

	err := stmt.QueryRow(connectionId).Scan(&pidId)
	if err != nil {
		return errorutil.Wrap(err)
	}

	err = decrementPidUsage(tx, trackerStmts, pidId)
	if err != nil {
		return errorutil.Wrap(err)
	}

	countConnections, err := countPidUsage(tx, trackerStmts, pidId)
	if err != nil {
		return errorutil.Wrap(err)
	}

	if countConnections > 0 {
		return nil
	}

	stmt = tx.Stmt(trackerStmts[deletePidById])

	defer func() {
		errorutil.MustSucceed(stmt.Close())
	}()

	_, err = stmt.Exec(pidId)
	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func deleteQueue(tx *sql.Tx, trackerStmts trackerStmts, queueId int64) error {
	// delete the queue itself
	stmt := tx.Stmt(trackerStmts[deleteQueueById])

	defer func() {
		errorutil.MustSucceed(stmt.Close())
	}()

	queueResult, err := stmt.Exec(queueId)
	if err != nil {
		return errorutil.Wrap(err)
	}

	_, err = queueResult.RowsAffected()
	if err != nil {
		return errorutil.Wrap(err)
	}

	// delete all data
	stmt = tx.Stmt(trackerStmts[deleteQueueDataById])

	defer func() {
		errorutil.MustSucceed(stmt.Close())
	}()

	_, err = stmt.Exec(queueId)
	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func incrementQueueUsage(tx *sql.Tx, trackerStmts trackerStmts, queueId int64) error {
	stmt := tx.Stmt(trackerStmts[incrementQueueUsageById])

	defer func() {
		errorutil.MustSucceed(stmt.Close())
	}()

	_, err := stmt.Exec(queueId)
	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

var ErrInvalidAffectedLines = errors.New(`Wrong number of affected lines`)

func decrementQueueUsage(tx *sql.Tx, trackerStmts trackerStmts, queueId int64) error {
	stmt := tx.Stmt(trackerStmts[decrementQueueUsageById])

	defer func() {
		errorutil.MustSucceed(stmt.Close())
	}()

	r, err := stmt.Exec(queueId)
	if err != nil {
		return errorutil.Wrap(err)
	}

	a, err := r.RowsAffected()
	if err != nil {
		return errorutil.Wrap(err)
	}

	if a != 1 {
		return ErrInvalidAffectedLines
	}

	return nil
}
