package tracking

import (
	"database/sql"
	"errors"
	"gitlab.com/lightmeter/controlcenter/data"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/pkg/runner"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
)

type queueInfo struct {
	queueId int64

	// a helper for debugging!
	loc data.RecordLocation
}

type queuesCommitNotifier struct {
	runner.CancelableRunner
	queuesToNotify <-chan queueInfo
	publisher      ResultPublisher
}

type notifierStmtKey uint

const (
	//nolint
	firstNotifierStmtKey notifierStmtKey = iota

	selectParentingQueueByNewQueue
	selectAllMessageIdsByQueue
	selectPidHostByQueue
	selectKeyValueFromConnections
	selectKeyValueFromQueues
	selectKeyValueFromQueuesByKeyType
	selectResultsByQueue
	selectKeyValueForResults
	selectQueryNameById

	lastNotifierStmtKey
)

var notifierStmtsText = map[notifierStmtKey]string{
	selectParentingQueueByNewQueue: `
		select
			orig_queue_id, parenting_type
		from
			queue_parenting
		where
			new_queue_id = ?
		group by
			orig_queue_id, new_queue_id, parenting_type`,
	selectAllMessageIdsByQueue: `
			select
				messageids.value
			from
				messageids join queues on queues.messageid_id == messageids.rowid
			where
				queues.rowid = ?`,
	selectPidHostByQueue: `
			select
				pids.host
			from
				queues join connections on queues.connection_id == connections.rowid
				join pids on connections.pid_id == pids.rowid
			where
				queues.rowid == ?`,
	selectKeyValueFromConnections: `
			select
				key, value
			from
				connection_data join connections on connection_data.connection_id = connections.rowid
				join queues on queues.connection_id == connections.rowid
			where
				queues.rowid = ?`,
	selectKeyValueFromQueues: `
			select
				queue_data.key, queue_data.value
			from
				queue_data join queues on queue_data.queue_id = queues.rowid
			where
				queues.rowid = ?`,
	selectKeyValueFromQueuesByKeyType: `
			select
				queue_data.key, queue_data.value
			from
				queue_data join queues on queue_data.queue_id = queues.rowid
			where
				queues.rowid = ? and queue_data.key = ?`,
	selectResultsByQueue: `
			select
				results.rowid
			from
				results join queues on results.queue_id == queues.rowid
			where
				queues.rowid == ?`,
	selectKeyValueForResults: `
				select
					result_data.key, result_data.value
				from
					result_data join results on result_data.result_id = results.rowid
				where
					results.rowid == ?`,
	selectQueryNameById: `select queue from queues where rowid = ?`,
}

type notifierStmts [lastNotifierStmtKey]*sql.Stmt

func prepareNotifierRoStmts(conn dbconn.RoConn) (notifierStmts, error) {
	stmts := notifierStmts{}

	for k, v := range notifierStmtsText {
		stmt, err := conn.Prepare(v)
		if err != nil {
			return notifierStmts{}, errorutil.Wrap(err)
		}

		stmts[k] = stmt
	}

	return stmts, nil
}

func findOrigQueueForQueueParenting(stmts notifierStmts, queueId int64) (int64, queueParentingType, error) {
	// first try to obtain the original queue which is not caused by a bounce
	var (
		origQueue     int64
		parentingType queueParentingType
	)

	err := stmts[selectParentingQueueByNewQueue].QueryRow(queueId).Scan(&origQueue, &parentingType)

	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, 0, errorutil.Wrap(err)
	}

	if err == nil {
		return origQueue, parentingType, nil
	}

	return origQueue, parentingType, errorutil.Wrap(err)
}

func findConnectionAndDeliveryQueue(stmts notifierStmts, queueId int64, loc data.RecordLocation) (connQueueId int64, deliveryQueueId int64, err error) {
	origQueue, parentingType, err := findOrigQueueForQueueParenting(stmts, queueId)

	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, 0, errorutil.Wrap(err)
	}

	if err != nil && errors.Is(err, sql.ErrNoRows) {
		// no parenting. the passed queue was used for both connection and delivery
		// TODO: investigate if this is really the case or if some information is missing!
		return queueId, queueId, nil
	}

	if parentingType == queueParentingRelayType {
		return origQueue, queueId, nil
	}

	// this is a bounce parenting relationship. I need to find the original one from it.
	// This is a, ugly recursive call, but we expect it to be execute at most twice, so that's ok.
	return findConnectionAndDeliveryQueue(stmts, origQueue, loc)
}

// FIXME: this method is way long. Really. It deserves urgentrefactoring
func runQueuesCommitNotifier(stmts notifierStmts, n *queuesCommitNotifier) error {
	for queueInfo := range n.queuesToNotify {
		queueId := queueInfo.queueId

		// find connection queue and delivery queue
		connQueueId, deliveryQueueId, err := findConnectionAndDeliveryQueue(stmts, queueId, queueInfo.loc)
		if err != nil {
			return errorutil.Wrap(err)
		}

		var messageId string

		err = stmts[selectAllMessageIdsByQueue].QueryRow(deliveryQueueId).Scan(&messageId)

		if err != nil {
			return errorutil.Wrap(err)
		}

		var deliveryServer string

		err = stmts[selectPidHostByQueue].QueryRow(deliveryQueueId).Scan(&deliveryServer)

		if err != nil {
			return errorutil.Wrap(err)
		}

		collectKeyValueResult := func(result *Result, stmt *sql.Stmt, args ...interface{}) error {
			var (
				key   int
				value interface{}
			)

			// fetch all results for connection
			rows, err := stmt.Query(args...)

			if err != nil {
				return errorutil.Wrap(err)
			}

			for rows.Next() {
				err = rows.Scan(&key, &value)

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

		connectionResult := Result{}

		err = collectKeyValueResult(&connectionResult, stmts[selectKeyValueFromConnections], connQueueId)

		if err != nil {
			return errorutil.Wrap(err)
		}

		queueResult := Result{}

		err = collectKeyValueResult(&queueResult, stmts[selectKeyValueFromQueues], connQueueId)

		if err != nil {
			return errorutil.Wrap(err)
		}

		deliveryQueueResult := Result{}

		err = collectKeyValueResult(&deliveryQueueResult, stmts[selectKeyValueFromQueuesByKeyType], deliveryQueueId, QueueOriginalMessageSizeKey)
		if err != nil {
			return errorutil.Wrap(err)
		}

		deliveryQueueResult[QueueProcessedMessageSizeKey] = deliveryQueueResult[QueueOriginalMessageSizeKey]
		deliveryQueueResult[QueueOriginalMessageSizeKey] = nil

		var deliveryQueueName string

		err = stmts[selectQueryNameById].QueryRow(deliveryQueueId).Scan(&deliveryQueueName)
		if err != nil {
			return errorutil.Wrap(err)
		}

		deliveryQueueResult[QueueDeliveryNameKey] = deliveryQueueName

		// iterate over all results
		resultsRows, err := stmts[selectResultsByQueue].Query(deliveryQueueId)

		if err != nil {
			return errorutil.Wrap(err)
		}

		var resultId int64

		for resultsRows.Next() {
			err = resultsRows.Scan(&resultId)

			if err != nil {
				return errorutil.Wrap(err)
			}

			resultResult := Result{}

			err = collectKeyValueResult(&resultResult, stmts[selectKeyValueForResults], resultId)

			if err != nil {
				return errorutil.Wrap(err)
			}

			mergedResults := mergeResults(resultResult, queueResult, connectionResult, deliveryQueueResult)

			mergedResults[QueueMessageIDKey] = messageId
			mergedResults[ResultDeliveryServerKey] = deliveryServer

			n.publisher.Publish(mergedResults)
		}

		if err := resultsRows.Err(); err != nil {
			return errorutil.Wrap(err)
		}
	}

	return nil
}

func mergeResults(results ...Result) Result {
	m := Result{}

	// TODO: consider rewritting this loop to be cache friendlier (by iterating on the same index in all arrays)
	for _, r := range results {
		for i, v := range r {
			if v != nil {
				m[i] = v
			}
		}
	}

	return m
}
