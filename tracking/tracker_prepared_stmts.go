package tracking

import (
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
)

type trackerStmtKey uint

const (
	//nolint
	firstTrackerStmtKey trackerStmtKey = iota

	insertPidOnConnection
	insertConnectionOnConnection
	insertConnectionDataTwoRows
	insertConnectionData
	selectConnectionAndUsageCounterForPid
	insertQueueForConnection
	incrementQueueUsageById
	decrementQueueUsageById
	queueUsageCounter
	insertQueueData
	selectMessageIdForMessage
	insertMessageId
	incrementMessageIdUsageById
	decrementMessageIdUsageById
	updateQueueWithMessageId
	selectQueueIdForQueue
	insertQueueDataFourRows
	insertQueueDataTwoRows
	insertQueueParenting
	insertNotificationQueue
	selectNewQueueFromParenting
	countNewQueueFromParenting
	selectQueueFromParentingNewQueue
	deleteQueueParentingById
	selectQueueById
	insertResultData15Rows
	insertResultData3Rows
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
	countWithMessageIdUsageById
	deleteMessageId
	incrementConnectionUsageById
	decrementConnectionUsageById
	countPidUsageById
	countConnectionUsageById
	countPidUsageByPidId
	incrementPidUsageById
	decrementPidUsageById

	lastTrackerStmtKey
)

var trackerStmtsText = map[trackerStmtKey]string{
	insertPidOnConnection:        `insert into pids(pid, host, usage_counter) values(?, ?, 0)`,
	insertConnectionOnConnection: `insert into connections(pid_id, usage_counter) values(?, 0)`,
	insertConnectionDataTwoRows:  `insert into connection_data(connection_id, key, value) values(?, ?, ?), (?, ?, ?)`,
	insertConnectionData:         `insert into connection_data(connection_id, key, value) values(?, ?, ?)`,
	selectConnectionAndUsageCounterForPid: `select
		connections.id, connections.usage_counter
	from
		connections join pids
	where
		connections.pid_id = pids.id 
		and pids.host = ? and pids.pid = ?`,
	// when a queue is created, the tracker is using it, therefore its counter is 1
	insertQueueForConnection:    `insert into queues(connection_id, queue, usage_counter) values(?, ?, 1)`,
	incrementQueueUsageById:     `update queues set usage_counter = usage_counter + 1 where id = ?`,
	decrementQueueUsageById:     `update queues set usage_counter = usage_counter - 1 where id = ?`,
	queueUsageCounter:           `select usage_counter from queues where id = ?`,
	insertQueueData:             `insert into queue_data(queue_id, key, value) values(?, ?, ?)`,
	selectMessageIdForMessage:   `select id from messageids where value = ?`,
	insertMessageId:             `insert into messageids(value, usage_counter) values(?, 1)`,
	incrementMessageIdUsageById: `update messageids set usage_counter = usage_counter + 1 where id = ?`,
	decrementMessageIdUsageById: `update messageids set usage_counter = usage_counter - 1 where id = ?`,
	updateQueueWithMessageId:    `update queues set messageid_id = ? where queues.id = ?`,
	selectQueueIdForQueue: `select
		queues.id
	from
		queues join connections on queues.connection_id = connections.id
		join pids on connections.pid_id = pids.id
	where
		pids.host = ? and queues.queue = ?`,
	insertQueueDataFourRows:          `insert into queue_data(queue_id, key, value) values(?, ?, ?), (?, ?, ?), (?, ?, ?), (?, ?, ?)`,
	insertQueueDataTwoRows:           `insert into queue_data(queue_id, key, value) values(?, ?, ?), (?, ?, ?)`,
	insertQueueParenting:             `insert into queue_parenting(orig_queue_id, new_queue_id, parenting_type) values(?, ?, ?)`,
	insertNotificationQueue:          `insert into notification_queues(result_id, filename, line) values(?, ?, ?)`,
	selectNewQueueFromParenting:      `select new_queue_id from queue_parenting where orig_queue_id = ?`,
	countNewQueueFromParenting:       `select count(new_queue_id) from queue_parenting where orig_queue_id = ?`,
	selectQueueFromParentingNewQueue: `select id, orig_queue_id from queue_parenting where new_queue_id = ?`,
	deleteQueueParentingById:         `delete from queue_parenting where id = ?`,
	selectQueueById:                  `select queue from queues where id = ?`,
	insertResultData15Rows: `insert into result_data(result_id, key, value)
		values(?, ?, ?),
					(?, ?, ?),
					(?, ?, ?),
					(?, ?, ?),
					(?, ?, ?),
					(?, ?, ?),
					(?, ?, ?),
					(?, ?, ?),
					(?, ?, ?),
					(?, ?, ?),
					(?, ?, ?),
					(?, ?, ?),
					(?, ?, ?),
					(?, ?, ?),
					(?, ?, ?)`,
	insertResultData3Rows: `insert into result_data(result_id, key, value)
		values(?, ?, ?),
					(?, ?, ?),
					(?, ?, ?)`,
	insertResult:                       `insert into results(queue_id) values(?)`,
	selectFromNotificationQueues:       `select id, result_id, filename, line from notification_queues`,
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
	countWithMessageIdUsageById:        `select usage_counter from messageids where id = ?`,
	deleteMessageId:                    `delete from messageids where id = ?`,
	incrementConnectionUsageById:       `update connections set usage_counter = usage_counter + 1 where id = ?`,
	decrementConnectionUsageById:       `update connections set usage_counter = usage_counter - 1 where id = ?`,
	countPidUsageById:                  `select usage_counter from pids where id = ?`,
	countConnectionUsageById:           `select usage_counter from connections where id = ?`,
	countPidUsageByPidId:               `select usage_counter from pids where id = ?`,
	incrementPidUsageById:              `update pids set usage_counter = usage_counter + 1 where id = ?`,
	decrementPidUsageById:              `update pids set usage_counter = usage_counter - 1 where id = ?`,
}

// TODO: close such statements when the tracker is deleted!!!
func prepareTrackerRwStmts(conn dbconn.RwConn) (trackerStmts, error) {
	stmts := trackerStmts{}

	for k, v := range trackerStmtsText {
		//nolint:sqlclosecheck
		stmt, err := conn.Prepare(v)
		if err != nil {
			return trackerStmts{}, errorutil.Wrap(err)
		}

		stmts[k] = stmt
	}

	return stmts, nil
}
