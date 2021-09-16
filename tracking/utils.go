// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package tracking

import (
	"database/sql"
	"errors"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
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

func tryToDeleteQueue(trackerStmts dbconn.TxPreparedStmts, queueId int64, loc postfix.RecordLocation) (bool, error) {
	deleted, err := tryToDeleteQueueNotIgnoringErrors(trackerStmts, queueId, loc)

	// Treat deletion errors (some queries return "norows") differently for now...
	if err != nil && (errors.Is(err, sql.ErrNoRows) || errors.Is(err, ErrInvalidAffectedLines)) {
		return false, &DeletionError{Err: errorutil.Wrap(err), Loc: loc}
	}

	if err != nil {
		return false, errorutil.Wrap(err)
	}

	return deleted, nil
}

func tryToDeleteQueueNotIgnoringErrors(trackerStmts dbconn.TxPreparedStmts, queueId int64, loc postfix.RecordLocation) (bool, error) {
	err := decrementQueueUsage(trackerStmts, queueId)
	if err != nil {
		return false, errorutil.Wrap(err, loc)
	}

	var usageCounter int

	//nolint:sqlclosecheck
	err = trackerStmts.Get(queueUsageCounter).QueryRow(queueId).Scan(&usageCounter)
	if err != nil {
		return false, errorutil.Wrap(err, loc)
	}

	if usageCounter > 0 {
		return false, nil
	}

	// Check if there's any queue that depends on me.
	// In such scenario, I cannot be deleted yet
	var countDependentQueues int64

	//nolint:sqlclosecheck
	err = trackerStmts.Get(countNewQueueFromParenting).QueryRow(queueId).Scan(&countDependentQueues)
	if err != nil {
		return false, errorutil.Wrap(err, loc)
	}

	if countDependentQueues > 0 {
		return false, nil
	}

	err = deleteQueueRec(trackerStmts, queueId)
	if err != nil {
		return false, errorutil.Wrap(err, loc)
	}

	return true, nil
}

func deleteQueueRec(trackerStmts dbconn.TxPreparedStmts, queueId int64) error {
	//nolint:rowserrcheck,sqlclosecheck
	rows, err := trackerStmts.Get(selectQueueFromParentingNewQueue).Query(queueId)
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

		err = deleteQueueRec(trackerStmts, dependencyQueueId)
		if err != nil {
			return errorutil.Wrap(err)
		}

		//nolint:sqlclosecheck
		_, err = trackerStmts.Get(deleteQueueParentingById).Exec(id)
		if err != nil {
			return errorutil.Wrap(err)
		}
	}

	err = rows.Err()
	if err != nil {
		return errorutil.Wrap(err)
	}

	err = tryToDeleteConnectionForQueue(trackerStmts, queueId)
	if err != nil {
		return errorutil.Wrap(err)
	}

	err = deleteQueue(trackerStmts, queueId)
	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func countQueuesOnConnection(trackerStmts dbconn.TxPreparedStmts, connectionId int64) (int, error) {
	var connectionCounter int

	//nolint:sqlclosecheck
	err := trackerStmts.Get(countConnectionUsageById).QueryRow(connectionId).Scan(&connectionCounter)
	if err != nil {
		return 0, errorutil.Wrap(err)
	}

	return connectionCounter, nil
}

func tryToDeleteConnectionForQueue(trackerStmts dbconn.TxPreparedStmts, queueId int64) error {
	var connectionId int64

	//nolint:sqlclosecheck
	err := trackerStmts.Get(connectionIdForQueue).QueryRow(queueId).Scan(&connectionId)
	if err != nil {
		return errorutil.Wrap(err)
	}

	err = decrementConnectionUsage(trackerStmts, connectionId)
	if err != nil {
		return errorutil.Wrap(err)
	}

	connectionCounter, err := countQueuesOnConnection(trackerStmts, connectionId)
	if err != nil {
		return errorutil.Wrap(err)
	}

	if connectionCounter > 0 {
		return nil
	}

	// TODO: do not delete connection if it's still active (no disconnect command has been done for it)

	err = deleteConnection(trackerStmts, connectionId)
	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func deleteConnection(trackerStmts dbconn.TxPreparedStmts, connectionId int64) error {
	err := tryToDeletePidForConnection(trackerStmts, connectionId)
	if err != nil {
		return errorutil.Wrap(err)
	}

	//nolint:sqlclosecheck
	_, err = trackerStmts.Get(deleteConnectionDataByConnectionId).Exec(connectionId)
	if err != nil {
		return errorutil.Wrap(err)
	}

	//nolint:sqlclosecheck
	_, err = trackerStmts.Get(deleteConnectionById).Exec(connectionId)
	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func countPidUsage(trackerStmts dbconn.TxPreparedStmts, pidId int64) (int, error) {
	var count int

	//nolint:sqlclosecheck
	err := trackerStmts.Get(countPidUsageByPidId).QueryRow(pidId).Scan(&count)
	if err != nil {
		return 0, errorutil.Wrap(err)
	}

	return count, nil
}

func incrementPidUsage(stmts dbconn.TxPreparedStmts, pidId int64) error {
	//nolint:sqlclosecheck
	_, err := stmts.Get(incrementPidUsageById).Exec(pidId)
	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func decrementPidUsage(stmts dbconn.TxPreparedStmts, pidId int64) error {
	//nolint:sqlclosecheck
	_, err := stmts.Get(decrementPidUsageById).Exec(pidId)
	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func tryToDeletePidForConnection(trackerStmts dbconn.TxPreparedStmts, connectionId int64) error {
	var pidId int64

	//nolint:sqlclosecheck
	err := trackerStmts.Get(pidIdForConnection).QueryRow(connectionId).Scan(&pidId)
	if err != nil {
		return errorutil.Wrap(err)
	}

	err = decrementPidUsage(trackerStmts, pidId)
	if err != nil {
		return errorutil.Wrap(err)
	}

	countConnections, err := countPidUsage(trackerStmts, pidId)
	if err != nil {
		return errorutil.Wrap(err)
	}

	if countConnections > 0 {
		return nil
	}

	//nolint:sqlclosecheck
	_, err = trackerStmts.Get(deletePidById).Exec(pidId)
	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func deleteQueue(trackerStmts dbconn.TxPreparedStmts, queueId int64) error {
	// delete the queue itself
	//nolint:sqlclosecheck
	queueResult, err := trackerStmts.Get(deleteQueueById).Exec(queueId)
	if err != nil {
		return errorutil.Wrap(err)
	}

	_, err = queueResult.RowsAffected()
	if err != nil {
		return errorutil.Wrap(err)
	}

	// delete all data
	//nolint:sqlclosecheck
	_, err = trackerStmts.Get(deleteQueueDataById).Exec(queueId)
	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func incrementQueueUsage(trackerStmts dbconn.TxPreparedStmts, queueId int64) error {
	//nolint:sqlclosecheck
	_, err := trackerStmts.Get(incrementQueueUsageById).Exec(queueId)
	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

var ErrInvalidAffectedLines = errors.New(`Wrong number of affected lines`)

func decrementQueueUsage(trackerStmts dbconn.TxPreparedStmts, queueId int64) error {
	//nolint:sqlclosecheck
	r, err := trackerStmts.Get(decrementQueueUsageById).Exec(queueId)
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
