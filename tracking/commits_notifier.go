// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package tracking

import (
	"database/sql"
	"errors"
	"gitlab.com/lightmeter/controlcenter/data"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/pkg/runner"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
)

type resultInfo struct {
	id int64

	// a helper for debugging!
	loc data.RecordLocation
}

type resultsNotifier struct {
	runner.CancelableRunner
	resultsToNotify <-chan resultInfos
	publisher       ResultPublisher
}

type notifierStmtKey uint

const (
	//nolint
	firstNotifierStmtKey notifierStmtKey = iota

	selectParentingQueueByNewQueue
	selectAllMessageIdsByQueue
	selectQueueIdFromResult
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
				messageids.value, messageids.filename, messageids.line
			from
				messageids join queues on queues.messageid_id == messageids.id
			where
				queues.id = ?`,
	selectQueueIdFromResult: `select queue_id from results where id = ?`,
	selectPidHostByQueue: `
			select
				pids.host
			from
				queues join connections on queues.connection_id == connections.id
				join pids on connections.pid_id == pids.id
			where
				queues.id == ?`,
	selectKeyValueFromConnections: `
			select
				connection_data.id, key, value
			from
				connection_data join connections on connection_data.connection_id = connections.id
				join queues on queues.connection_id == connections.id
			where
				queues.id = ?`,
	selectKeyValueFromQueues: `
			select
				queue_data.key, queue_data.key, queue_data.value
			from
				queue_data join queues on queue_data.queue_id = queues.id
			where
				queues.id = ?`,
	selectKeyValueFromQueuesByKeyType: `
			select
				queue_data.id, queue_data.key, queue_data.value
			from
				queue_data join queues on queue_data.queue_id = queues.id
			where
				queues.id = ? and queue_data.key = ?`,
	selectResultsByQueue: `
			select
				results.id
			from
				results join queues on results.queue_id == queues.id
			where
				queues.id == ?`,
	selectKeyValueForResults: `
				select
					result_data.id, result_data.key, result_data.value
				from
					result_data join results on result_data.result_id = results.id
				where
					results.id == ?`,
	selectQueryNameById: `select queue from queues where id = ?`,
}

type notifierStmts [lastNotifierStmtKey]*sql.Stmt

func prepareNotifierRoStmts(conn dbconn.RoConn) (notifierStmts, error) {
	stmts := notifierStmts{}

	for k, v := range notifierStmtsText {
		// The prepared queries are closed when the application ends
		//nolint:sqlclosecheck
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

func collectConnectionKeyValueResults(stmts notifierStmts, queueId int64) (Result, error) {
	result := Result{}

	if err := collectKeyValueResult(&result, stmts[selectKeyValueFromConnections], queueId); err != nil {
		return Result{}, errorutil.Wrap(err)
	}

	return result, nil
}

func collectQueuesKeyValueResults(stmts notifierStmts, queueId int64) (Result, error) {
	result := Result{}

	if err := collectKeyValueResult(&result, stmts[selectKeyValueFromQueues], queueId); err != nil {
		return Result{}, errorutil.Wrap(err)
	}

	return result, nil
}

func collectQueuesOriginalMessageSizeKeyValueResults(stmts notifierStmts, queueId int64) (Result, error) {
	result := Result{}

	if err := collectKeyValueResult(&result, stmts[selectKeyValueFromQueuesByKeyType], queueId, QueueOriginalMessageSizeKey); err != nil {
		return Result{}, errorutil.Wrap(err)
	}

	return result, nil
}

func collectResultKeyValueResults(stmts notifierStmts, resultId int64) (Result, error) {
	result := Result{}

	if err := collectKeyValueResult(&result, stmts[selectKeyValueForResults], resultId); err != nil {
		return Result{}, errorutil.Wrap(err)
	}

	return result, nil
}

// FIXME: this method is way too long. Really. It deserves urgent refactoring
func buildAndPublishResult(stmts notifierStmts,
	resultId int64,
	pub ResultPublisher,
	trackerStmts trackerStmts,
	actions chan<- func(*sql.Tx) error) error {
	var queueId int64

	resultResult, err := collectResultKeyValueResults(stmts, resultId)
	if err != nil {
		return errorutil.Wrap(err)
	}

	resultInfo := resultInfo{
		id: resultId,
		loc: data.RecordLocation{
			Line:     uint64(resultResult[ResultDeliveryFileLineKey].(int64)),
			Filename: resultResult[ResultDeliveryFilenameKey].(string),
		},
	}

	err = stmts[selectQueueIdFromResult].QueryRow(resultId).Scan(&queueId)
	if err != nil {
		return errorutil.Wrap(err)
	}

	// find connection queue and delivery queue
	connQueueId, deliveryQueueId, err := findConnectionAndDeliveryQueue(stmts, queueId, resultInfo.loc)
	if err != nil {
		return errorutil.Wrap(err)
	}

	var (
		messageId         string
		messageIdFilename string
		messageIdLine     int64
	)

	err = stmts[selectAllMessageIdsByQueue].QueryRow(deliveryQueueId).Scan(&messageId, &messageIdFilename, &messageIdLine)
	if err != nil {
		return errorutil.Wrap(err)
	}

	var deliveryServer string

	err = stmts[selectPidHostByQueue].QueryRow(deliveryQueueId).Scan(&deliveryServer)
	if err != nil {
		return errorutil.Wrap(err)
	}

	// TODO: operate all on the same Result{}, except for the deliveryQueue stuff.
	// It makes the mergeResults faster as it operates on less arrays!

	connResult, err := collectConnectionKeyValueResults(stmts, connQueueId)
	if err != nil {
		return errorutil.Wrap(err)
	}

	queueResult, err := collectQueuesKeyValueResults(stmts, connQueueId)
	if err != nil {
		return errorutil.Wrap(err)
	}

	deliveryQueueResult, err := collectQueuesOriginalMessageSizeKeyValueResults(stmts, deliveryQueueId)
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

	mergedResults := mergeResults(resultResult, queueResult, connResult, deliveryQueueResult)

	mergedResults[MessageIdFilenameKey] = messageIdFilename
	mergedResults[MessageIdLineKey] = messageIdLine
	mergedResults[QueueMessageIDKey] = messageId
	mergedResults[ResultDeliveryServerKey] = deliveryServer

	pub.Publish(mergedResults)

	deleteQueueAction := func(tx *sql.Tx) error {
		_, err := tryToDeleteQueue(tx, trackerStmts, queueId, resultInfo.loc)
		if err != nil {
			return errorutil.Wrap(err)
		}

		return nil
	}

	deleteResultAction := func(tx *sql.Tx) error {
		stmt := tx.Stmt(trackerStmts[deleteResultByIdKey])

		defer func() {
			errorutil.MustSucceed(stmt.Close())
		}()

		_, err := stmt.Exec(resultInfo.id)
		if err != nil {
			return errorutil.Wrap(err)
		}

		stmt = tx.Stmt(trackerStmts[deleteResultDataByResultId])

		defer func() {
			errorutil.MustSucceed(stmt.Close())
		}()

		_, err = stmt.Exec(resultInfo.id)
		if err != nil {
			return errorutil.Wrap(err)
		}

		return nil
	}

	actions <- func(tx *sql.Tx) error {
		if err := deleteResultAction(tx); err != nil {
			return errorutil.Wrap(err, resultInfo.loc)
		}

		if err := deleteQueueAction(tx); err != nil {
			return errorutil.Wrap(err, resultInfo.loc)
		}

		return nil
	}

	return nil
}

func runResultsNotifier(stmts notifierStmts, n *resultsNotifier, trackerStmts trackerStmts, actions chan<- func(*sql.Tx) error) error {
	for resultInfos := range n.resultsToNotify {
		for i := uint(0); i < resultInfos.size; i++ {
			resultInfo := resultInfos.values[i]

			err := buildAndPublishResult(stmts, resultInfo, n.publisher, trackerStmts, actions)

			if err == nil {
				continue
			}

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
