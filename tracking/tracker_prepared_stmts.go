// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package tracking

import (
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
)

const (
	insertPidOnConnection = iota
	insertConnectionOnConnection
	insertConnectionDataFourRows
	insertConnectionData
	selectConnectionAndUsageCounterForPid
	insertQueueForConnection
	incrementQueueUsageById
	decrementQueueUsageById
	queueUsageCounter
	insertQueueData
	insertResultData
	updateQueueWithMessageId
	selectQueueIdForQueue
	insertQueueParenting
	insertNotificationQueue
	countNewQueueFromParenting
	selectQueueFromParentingNewQueue
	deleteQueueParentingById
	selectQueueById
	insertResult
	deleteFromNotificationQueues
	selectFromNotificationQueues
	deleteResultByIdKey
	deleteResultDataByResultId
	deleteQueueDataById
	deleteQueueById
	deleteConnectionDataByConnectionId
	connectionIdForQueue
	deleteConnectionById
	pidIdForConnection
	deletePidById
	messageIdForQueue
	incrementConnectionUsageById
	decrementConnectionUsageById
	countPidUsageById
	countConnectionUsageById
	countPidUsageByPidId
	incrementPidUsageById
	decrementPidUsageById
	selectPidForPidAndHost
	selectConnectionAuthCountForQueue
	insertPreNotificationByQueueIdAndResultId
	selectPreNotificationResultIdsForQueue
	deletePreNotificationEntryByQueueId

	lastTrackerStmtKey
)

var trackerStmtsText = dbconn.StmtsText{
	insertPidOnConnection:        `insert into pids(pid, host, usage_counter) values(?, ?, 1)`,
	insertConnectionOnConnection: `insert into connections(pid_id, usage_counter) values(?, 0)`,
	insertConnectionDataFourRows: `insert into connection_data(connection_id, key, value) values(?, ?, ?), (?, ?, ?), (?, ?, ?), (?, ?, ?)`,
	insertConnectionData:         `insert into connection_data(connection_id, key, value) values(?, ?, ?)`,
	selectConnectionAndUsageCounterForPid: `select
		connections.id, connections.usage_counter
	from
		connections join pids
	where
		connections.pid_id = pids.id 
		and pids.host = ? and pids.pid = ?
	order by
		connections.id desc
	limit 1
	`,
	// when a queue is created, the tracker is using it, therefore its counter is 1
	insertQueueForConnection: `insert into queues(connection_id, queue, usage_counter) values(?, ?, 1)`,
	incrementQueueUsageById:  `update queues set usage_counter = usage_counter + 1 where id = ?`,
	decrementQueueUsageById:  `update queues set usage_counter = usage_counter - 1 where id = ?`,
	queueUsageCounter:        `select usage_counter from queues where id = ?`,
	insertQueueData:          `insert into queue_data(queue_id, key, value) values(?, ?, ?)`,
	insertResultData:         `insert into result_data(result_id, key, value) values(?, ?, ?)`,
	updateQueueWithMessageId: `update queues set messageid_id = ? where queues.id = ?`,
	selectQueueIdForQueue: `select
		queues.id
	from
		queues join connections on queues.connection_id = connections.id
		join pids on connections.pid_id = pids.id
	where
		queues.queue = ?`,
	insertQueueParenting: `insert into queue_parenting(orig_queue_id, new_queue_id, parenting_type) values(?, ?, ?)`,
	// TODO: perform a migration that remove filename and line fields
	insertNotificationQueue:            `insert into notification_queues(result_id, filename, line) values(?, '', 0)`,
	countNewQueueFromParenting:         `select count(new_queue_id) from queue_parenting where orig_queue_id = ?`,
	selectQueueFromParentingNewQueue:   `select id, orig_queue_id from queue_parenting where new_queue_id = ?`,
	deleteQueueParentingById:           `delete from queue_parenting where id = ?`,
	selectQueueById:                    `select queue from queues where id = ?`,
	insertResult:                       `insert into results(queue_id) values(?)`,
	selectFromNotificationQueues:       `select id, result_id from notification_queues`,
	deleteFromNotificationQueues:       `delete from notification_queues where id = ?`,
	deleteResultByIdKey:                `delete from results where id = ?`,
	deleteResultDataByResultId:         `delete from result_data where result_id = ?`,
	deleteQueueDataById:                `delete from queue_data where queue_id = ?`,
	deleteQueueById:                    `delete from queues where id = ?`,
	deleteConnectionDataByConnectionId: `delete from connection_data where connection_id = ?`,
	connectionIdForQueue:               `select connection_id from queues where id = ?`,
	deleteConnectionById:               `delete from connections where id = ?`,
	pidIdForConnection:                 `select pid_id from connections where id = ?`,
	deletePidById:                      `delete from pids where id = ?`,
	messageIdForQueue:                  `select messageid_id from queues where id = ?`,
	incrementConnectionUsageById:       `update connections set usage_counter = usage_counter + 1 where id = ?`,
	decrementConnectionUsageById:       `update connections set usage_counter = usage_counter - 1 where id = ?`,
	countPidUsageById:                  `select usage_counter from pids where id = ?`,
	countConnectionUsageById:           `select usage_counter from connections where id = ?`,
	countPidUsageByPidId:               `select usage_counter from pids where id = ?`,
	incrementPidUsageById:              `update pids set usage_counter = usage_counter + 1 where id = ?`,
	decrementPidUsageById:              `update pids set usage_counter = usage_counter - 1 where id = ?`,
	selectPidForPidAndHost:             `select id from pids where pid = ? and host = ?`,
	selectConnectionAuthCountForQueue: `select 
	connection_data.value
from
	queues join connections on queues.connection_id = connections.id
		join connection_data on connection_data.connection_id = connections.id
where
	queues.id = ?
	and connection_data.key = ?`,
	insertPreNotificationByQueueIdAndResultId: `insert into prenotification_results(queue_id, result_id) values(?, ?)`,
	selectPreNotificationResultIdsForQueue:    `select result_id from prenotification_results where queue_id = ?`,
	deletePreNotificationEntryByQueueId:       `delete from prenotification_results where queue_id = ?`,
}
