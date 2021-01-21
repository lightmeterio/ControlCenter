package tracking

import (
	"database/sql"
	"errors"
	"gitlab.com/lightmeter/controlcenter/data"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
)

func collectKeyValueResult(result *Result, stmt *sql.Stmt, args ...interface{}) error {
	var (
		id    int64
		key   int
		value interface{}
	)

	rows, err := stmt.Query(args...)
	if err != nil {
		return errorutil.Wrap(err)
	}

	for rows.Next() {
		err = rows.Scan(&id, &key, &value)
		if err != nil {
			return errorutil.Wrap(err)
		}

		// TODO: abort if the key is not a valid result key (out of index)
		result[key] = value
	}

	if err := rows.Err(); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

type DeletionError struct {
	Err *errorutil.Error
	Loc data.RecordLocation
}

func (e *DeletionError) Unwrap() error {
	return e.Err
}

func (e *DeletionError) Error() string {
	return e.Err.Error()
}

func tryToDeleteQueue(tx *sql.Tx, trackerStmts trackerStmts, queueId int64, loc data.RecordLocation) (bool, error) {
	deleted, err := tryToDeleteQueueNotIgnoringErrors(tx, trackerStmts, queueId)

	// Treat deletion errors (some queries return "norows") differently for now...
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		return false, &DeletionError{Err: errorutil.Wrap(err), Loc: loc}
	}

	if err != nil {
		return false, errorutil.Wrap(err)
	}

	return deleted, nil
}

func tryToDeleteQueueNotIgnoringErrors(tx *sql.Tx, trackerStmts trackerStmts, queueId int64) (bool, error) {
	err := decrementQueueUsage(tx, trackerStmts, queueId)
	if err != nil {
		return false, errorutil.Wrap(err)
	}

	var usageCounter int

	err = tx.Stmt(trackerStmts[queueUsageCounter]).QueryRow(queueId).Scan(&usageCounter)
	if err != nil {
		return false, errorutil.Wrap(err)
	}

	if usageCounter > 0 {
		return false, nil
	}

	// Check if there's any queue that depends on me.
	// In such scenario, I cannot be deleted yet
	var countDependentQueues int64

	err = tx.Stmt(trackerStmts[countNewQueueFromParenting]).QueryRow(queueId).Scan(&countDependentQueues)
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
	rows, err := tx.Stmt(trackerStmts[selectQueueFromParentingNewQueue]).Query(queueId)
	if err != nil {
		return errorutil.Wrap(err)
	}

	var (
		dependencyQueueId int64
		rowid             int64
	)

	for rows.Next() {
		err = rows.Scan(&rowid, &dependencyQueueId)
		if err != nil {
			return errorutil.Wrap(err)
		}

		err = deleteQueueRec(tx, trackerStmts, dependencyQueueId)
		if err != nil {
			return errorutil.Wrap(err)
		}

		_, err = tx.Stmt(trackerStmts[deleteQueueParentingById]).Exec(rowid)
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

	err := tx.Stmt(trackerStmts[countConnectionUsageById]).QueryRow(connectionId).Scan(&connectionCounter)
	if err != nil {
		return 0, errorutil.Wrap(err)
	}

	return connectionCounter, nil
}

func tryToDeleteConnectionForQueue(tx *sql.Tx, trackerStmts trackerStmts, queueId int64) error {
	var connectionId int64

	err := tx.Stmt(trackerStmts[connectionIdForQueue]).QueryRow(queueId).Scan(&connectionId)
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

	_, err = tx.Stmt(trackerStmts[deleteConnectionDataByConnectionId]).Exec(connectionId)
	if err != nil {
		return errorutil.Wrap(err)
	}

	_, err = tx.Stmt(trackerStmts[deleteConnectionById]).Exec(connectionId)
	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func countPidUsage(tx *sql.Tx, trackerStmts trackerStmts, pidId int64) (int, error) {
	var count int

	err := tx.Stmt(trackerStmts[countPidUsageByPidId]).QueryRow(pidId).Scan(&count)
	if err != nil {
		return 0, errorutil.Wrap(err)
	}

	return count, nil
}

func incrementPidUsage(tx *sql.Tx, stmts trackerStmts, pidId int64) error {
	_, err := tx.Stmt(stmts[incrementPidUsageById]).Exec(pidId)
	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func decrementPidUsage(tx *sql.Tx, stmts trackerStmts, pidId int64) error {
	_, err := tx.Stmt(stmts[decrementPidUsageById]).Exec(pidId)
	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func tryToDeletePidForConnection(tx *sql.Tx, trackerStmts trackerStmts, connectionId int64) error {
	var pidId int64

	err := tx.Stmt(trackerStmts[pidIdForConnection]).QueryRow(connectionId).Scan(&pidId)
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

	_, err = tx.Stmt(trackerStmts[deletePidById]).Exec(pidId)
	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

//nolint:deadcode,unused
func getQueueName(tx *sql.Tx, queueId int64) (string, error) {
	var s string

	if err := tx.QueryRow(`select queue from queues where rowid = ?`, queueId).Scan(&s); err != nil {
		return "", errorutil.Wrap(err)
	}

	return s, nil
}

func deleteQueue(tx *sql.Tx, trackerStmts trackerStmts, queueId int64) error {
	if err := tryToDeleteQueueMessageId(tx, trackerStmts, queueId); err != nil {
		return errorutil.Wrap(err)
	}

	// delete the queue itself
	queueResult, err := tx.Stmt(trackerStmts[deleteQueueById]).Exec(queueId)
	if err != nil {
		return errorutil.Wrap(err)
	}

	_, err = queueResult.RowsAffected()
	if err != nil {
		return errorutil.Wrap(err)
	}

	// delete all data
	_, err = tx.Stmt(trackerStmts[deleteQueueDataById]).Exec(queueId)
	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func tryToDeleteQueueMessageId(tx *sql.Tx, trackerStmts trackerStmts, queueId int64) error {
	var messageId int64

	err := tx.Stmt(trackerStmts[messageIdForQueue]).QueryRow(queueId).Scan(&messageId)
	if err != nil {
		return errorutil.Wrap(err)
	}

	var queuesWithMessageIdCount int

	err = tx.Stmt(trackerStmts[countQueuesWithMessageId]).QueryRow(messageId).Scan(&queuesWithMessageIdCount)
	if err != nil {
		return errorutil.Wrap(err)
	}

	if queuesWithMessageIdCount > 1 {
		return nil
	}

	_, err = tx.Stmt(trackerStmts[deleteMessageId]).Exec(messageId)
	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func incrementQueueUsage(tx *sql.Tx, trackerStmts trackerStmts, queueId int64) error {
	_, err := tx.Stmt(trackerStmts[incrementQueueUsageById]).Exec(queueId)
	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func decrementQueueUsage(tx *sql.Tx, trackerStmts trackerStmts, queueId int64) error {
	_, err := tx.Stmt(trackerStmts[decrementQueueUsageById]).Exec(queueId)
	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}
