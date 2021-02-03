// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package tracking

import (
	"database/sql"
	"errors"
	"github.com/rs/zerolog/log"
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

type queuesCommitNotifier struct {
	runner.CancelableRunner
	resultsToNotify <-chan resultInfo
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
				messageids.value
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

// FIXME: this method is way too long. Really. It deserves urgent refactoring
func buildAndPublishResult(stmts notifierStmts,
	resultInfo resultInfo,
	pub ResultPublisher,
	trackerStmts trackerStmts,
	actions chan<- func(*sql.Tx) error) error {
	var queueId int64

	err := stmts[selectQueueIdFromResult].QueryRow(&resultInfo.id).Scan(&queueId)
	if err != nil {
		return errorutil.Wrap(err)
	}

	// find connection queue and delivery queue
	connQueueId, deliveryQueueId, err := findConnectionAndDeliveryQueue(stmts, queueId, resultInfo.loc)
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

	// TODO: operate all on the same Result{}, except for the deliveryQueue stuff.
	// It makes the mergeResults faster as it operates on less arrays!

	connResult := Result{}

	err = collectKeyValueResult(&connResult, stmts[selectKeyValueFromConnections], connQueueId)
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

	resultResult := Result{}

	err = collectKeyValueResult(&resultResult, stmts[selectKeyValueForResults], resultInfo.id)
	if err != nil {
		return errorutil.Wrap(err)
	}

	mergedResults := mergeResults(resultResult, queueResult, connResult, deliveryQueueResult)

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
		if err := deleteQueueAction(tx); err != nil {
			return errorutil.Wrap(err)
		}

		if err := deleteResultAction(tx); err != nil {
			return errorutil.Wrap(err)
		}

		return nil
	}

	return nil
}

func runResultsNotifier(stmts notifierStmts, n *queuesCommitNotifier, trackerStmts trackerStmts, actions chan<- func(*sql.Tx) error) error {
	for resultInfo := range n.resultsToNotify {
		err := buildAndPublishResult(stmts, resultInfo, n.publisher, trackerStmts, actions)

		if err == nil {
			continue
		}

		if errors.Is(err, sql.ErrNoRows) {
			//nolint:errorlint
			log.Warn().Msgf("Ignoring error notifying result: %v:%v, error: %v", resultInfo.loc.Filename, resultInfo.loc.Line, err.(*errorutil.Error).Chain())
			continue
		}

		return errorutil.Wrap(err)
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
