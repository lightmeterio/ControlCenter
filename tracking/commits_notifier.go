// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package tracking

import (
	"database/sql"
	"errors"
	"github.com/rs/zerolog/log"
	"gitlab.com/lightmeter/controlcenter/data"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/pkg/runner"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	//"time"
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
	counter         uint64
	id              int
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
				connection_data.id, connection_data.key, connection_data.value
			from
				connection_data join connections on connection_data.connection_id = connections.id
				join queues on queues.connection_id == connections.id
			where
				queues.id = ?`,
	selectKeyValueFromQueues: `
			select
				queue_data.id, queue_data.key, queue_data.value
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

func findOrigQueueForQueueParenting(conn *dbconn.RoPooledConn, queueId int64) (int64, queueParentingType, error) {
	// first try to obtain the original queue which is not caused by a bounce
	var (
		origQueue     int64
		parentingType queueParentingType
	)

	err := conn.Stmts[selectParentingQueueByNewQueue].QueryRow(queueId).Scan(&origQueue, &parentingType)

	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, 0, errorutil.Wrap(err)
	}

	if err == nil {
		return origQueue, parentingType, nil
	}

	return origQueue, parentingType, errorutil.Wrap(err)
}

func prepareCommitterConnection(conn *dbconn.RoPooledConn) error {
	for k, v := range notifierStmtsText {
		//nolint:sqlclosecheck
		stmt, err := conn.Prepare(v)
		if err != nil {
			return errorutil.Wrap(err)
		}

		conn.Stmts[k] = stmt

		conn.Closers.Add(stmt)
	}

	return nil
}

func findConnectionAndDeliveryQueue(conn *dbconn.RoPooledConn, queueId int64, loc data.RecordLocation) (connQueueId int64, deliveryQueueId int64, err error) {
	origQueue, parentingType, err := findOrigQueueForQueueParenting(conn, queueId)

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
	return findConnectionAndDeliveryQueue(conn, origQueue, loc)
}

func collectConnectionKeyValueResults(conn *dbconn.RoPooledConn, queueId int64) (Result, error) {
	result := Result{}

	if err := collectKeyValueResult(&result, conn.Stmts[selectKeyValueFromConnections], queueId); err != nil {
		return Result{}, errorutil.Wrap(err)
	}

	return result, nil
}

func collectQueuesKeyValueResults(conn *dbconn.RoPooledConn, queueId int64) (Result, error) {
	result := Result{}

	if err := collectKeyValueResult(&result, conn.Stmts[selectKeyValueFromQueues], queueId); err != nil {
		return Result{}, errorutil.Wrap(err)
	}

	return result, nil
}

func collectQueuesOriginalMessageSizeKeyValueResults(conn *dbconn.RoPooledConn, queueId int64) (Result, error) {
	result := Result{}

	if err := collectKeyValueResult(&result, conn.Stmts[selectKeyValueFromQueuesByKeyType], queueId, QueueOriginalMessageSizeKey); err != nil {
		return Result{}, errorutil.Wrap(err)
	}

	return result, nil
}

func collectResultKeyValueResults(conn *dbconn.RoPooledConn, resultId int64) (Result, error) {
	result := Result{}

	if err := collectKeyValueResult(&result, conn.Stmts[selectKeyValueForResults], resultId); err != nil {
		return Result{}, errorutil.Wrap(err)
	}

	return result, nil
}

// FIXME: this method is way too long. Really. It deserves urgent refactoring
func buildAndPublishResult(
	conn *dbconn.RoPooledConn,
	resultId int64,
	pub ResultPublisher,
	trackerStmts trackerStmts,
	actions *txActions) (resultInfo, error) {
	var queueId int64

	resultResult, err := collectResultKeyValueResults(conn, resultId)
	if err != nil {
		return resultInfo{}, errorutil.Wrap(err)
	}

	resultInfo := resultInfo{
		id: resultId,
		loc: data.RecordLocation{
			Line:     uint64(resultResult[ResultDeliveryFileLineKey].AsInt64),
			Filename: resultResult[ResultDeliveryFilenameKey].AsString,
		},
	}

	err = conn.Stmts[selectQueueIdFromResult].QueryRow(resultId).Scan(&queueId)
	if err != nil {
		return resultInfo, errorutil.Wrap(err, resultInfo.loc)
	}

	// find connection queue and delivery queue
	connQueueId, deliveryQueueId, err := findConnectionAndDeliveryQueue(conn, queueId, resultInfo.loc)
	if err != nil {
		return resultInfo, errorutil.Wrap(err, resultInfo.loc)
	}

	var (
		messageId         string
		messageIdFilename string
		messageIdLine     int64
	)

	err = conn.Stmts[selectAllMessageIdsByQueue].QueryRow(deliveryQueueId).Scan(&messageId, &messageIdFilename, &messageIdLine)
	if err != nil {
		return resultInfo, errorutil.Wrap(err, resultInfo.loc)
	}

	var deliveryServer string

	err = conn.Stmts[selectPidHostByQueue].QueryRow(deliveryQueueId).Scan(&deliveryServer)
	if err != nil {
		return resultInfo, errorutil.Wrap(err, resultInfo.loc)
	}

	// TODO: operate all on the same Result{}, except for the deliveryQueue stuff.
	// It makes the mergeResults faster as it operates on less arrays!

	connResult, err := collectConnectionKeyValueResults(conn, connQueueId)
	if err != nil {
		return resultInfo, errorutil.Wrap(err, resultInfo.loc)
	}

	queueResult, err := collectQueuesKeyValueResults(conn, connQueueId)
	if err != nil {
		return resultInfo, errorutil.Wrap(err, resultInfo.loc)
	}

	deliveryQueueResult, err := collectQueuesOriginalMessageSizeKeyValueResults(conn, deliveryQueueId)
	if err != nil {
		return resultInfo, errorutil.Wrap(err, resultInfo.loc)
	}

	deliveryQueueResult[QueueProcessedMessageSizeKey] = deliveryQueueResult[QueueOriginalMessageSizeKey]
	deliveryQueueResult[QueueOriginalMessageSizeKey] = ResultEntryNone()

	var deliveryQueueName string

	err = conn.Stmts[selectQueryNameById].QueryRow(deliveryQueueId).Scan(&deliveryQueueName)
	if err != nil {
		return resultInfo, errorutil.Wrap(err, resultInfo.loc)
	}

	deliveryQueueResult[QueueDeliveryNameKey] = ResultEntryString(deliveryQueueName)

	mergedResults := mergeResults(resultResult, queueResult, connResult, deliveryQueueResult)

	mergedResults[MessageIdFilenameKey] = ResultEntryString(messageIdFilename)
	mergedResults[MessageIdLineKey] = ResultEntryInt64(messageIdLine)
	mergedResults[QueueMessageIDKey] = ResultEntryString(messageId)
	mergedResults[ResultDeliveryServerKey] = ResultEntryString(deliveryServer)

	pub.Publish(mergedResults)

	actions.actions[actions.size] = func(tx *sql.Tx) error {
		if err := deleteResultAction(tx, trackerStmts, resultInfo); err != nil {
			return errorutil.Wrap(err, resultInfo.loc)
		}

		if err := deleteQueueAction(tx, trackerStmts, resultInfo, queueId); err != nil {
			return errorutil.Wrap(err, resultInfo.loc)
		}

		return nil
	}

	actions.size++

	return resultInfo, nil
}

func deleteQueueAction(tx *sql.Tx, trackerStmts trackerStmts, resultInfo resultInfo, queueId int64) error {
	_, err := tryToDeleteQueue(tx, trackerStmts, queueId, resultInfo.loc)
	if err != nil {
		return errorutil.Wrap(err, resultInfo.loc)
	}

	return nil
}

func deleteResultAction(tx *sql.Tx, trackerStmts trackerStmts, resultInfo resultInfo) error {
	stmt := tx.Stmt(trackerStmts[deleteResultByIdKey])

	defer func() {
		errorutil.MustSucceed(stmt.Close())
	}()

	_, err := stmt.Exec(resultInfo.id)
	if err != nil {
		return errorutil.Wrap(err, resultInfo.loc)
	}

	stmt = tx.Stmt(trackerStmts[deleteResultDataByResultId])

	defer func() {
		errorutil.MustSucceed(stmt.Close())
	}()

	_, err = stmt.Exec(resultInfo.id)
	if err != nil {
		return errorutil.Wrap(err, resultInfo.loc)
	}

	return nil
}

func runResultsNotifier(conn *dbconn.RoPooledConn, n *resultsNotifier, trackerStmts trackerStmts, actionsChan chan<- txActions) error {
	for resultInfos := range n.resultsToNotify {
		actions := txActions{}

		//log.Info().Msgf("Notifier %v started notifying batch %v:%v", n.id, resultInfos.batchId, resultInfos.id)

		//start := time.Now()

		for i := uint(0); i < resultInfos.size; i++ {
			id := resultInfos.values[i]

			resultInfo, err := buildAndPublishResult(conn, id, n.publisher, trackerStmts, &actions)

			if err == nil {
				continue
			}

			if errors.Is(err, sql.ErrNoRows) {
				log.Warn().Msgf("Ignoring error notifying result: %v:%v, error: %v", resultInfo.loc.Filename, resultInfo.loc.Line, err.(*errorutil.Error).Chain())
				continue
			}

			return errorutil.Wrap(err)
		}

		n.counter++

		//log.Info().Msgf("Notifier %v has notified %v actions in batch %v:%v in %v", n.id, actions.size, resultInfos.batchId, resultInfos.id, time.Now().Sub(start))

		actionsChan <- actions
	}

	return nil
}

func mergeResults(results ...Result) Result {
	m := Result{}

	// TODO: consider rewritting this loop to be cache friendlier (by iterating on the same index in all arrays)
	for _, r := range results {
		for i, v := range r {
			if v.Type != ResultEntryTypeNone {
				m[i] = v
			}
		}
	}

	return m
}
