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
		connections.rowid, connections.usage_counter
	from
		connections join pids
	where
		connections.pid_id = pids.rowid 
		and pids.host = ? and pids.pid = ?`,
	// when a queue is created, the tracker is using it, therefore its counter is 1
	insertQueueForConnection:    `insert into queues(connection_id, queue, usage_counter) values(?, ?, 1)`,
	incrementQueueUsageById:     `update queues set usage_counter = usage_counter + 1 where rowid = ?`,
	decrementQueueUsageById:     `update queues set usage_counter = usage_counter - 1 where rowid = ?`,
	queueUsageCounter:           `select usage_counter from queues where rowid = ?`,
	insertQueueData:             `insert into queue_data(queue_id, key, value) values(?, ?, ?)`,
	selectMessageIdForMessage:   `select rowid from messageids where value = ?`,
	insertMessageId:             `insert into messageids(value, usage_counter) values(?, 1)`,
	incrementMessageIdUsageById: `update messageids set usage_counter = usage_counter + 1 where rowid = ?`,
	decrementMessageIdUsageById: `update messageids set usage_counter = usage_counter - 1 where rowid = ?`,
	updateQueueWithMessageId:    `update queues set messageid_id = ? where queues.rowid = ?`,
	selectQueueIdForQueue: `select
		queues.rowid
	from
		queues join connections on queues.connection_id = connections.rowid
		join pids on connections.pid_id = pids.rowid
	where
		pids.host = ? and queues.queue = ?`,
	insertQueueDataFourRows: `insert into queue_data(queue_id, key, value)
		values(?, ?, ?),
					(?, ?, ?),
					(?, ?, ?),
					(?, ?, ?)`,
	insertQueueParenting:             `insert into queue_parenting(orig_queue_id, new_queue_id, parenting_type) values(?, ?, ?)`,
	insertNotificationQueue:          `insert into notification_queues(result_id, filename, line) values(?, ?, ?)`,
	selectNewQueueFromParenting:      `select new_queue_id from queue_parenting where orig_queue_id = ?`,
	countNewQueueFromParenting:       `select count(new_queue_id) from queue_parenting where orig_queue_id = ?`,
	selectQueueFromParentingNewQueue: `select rowid, orig_queue_id from queue_parenting where new_queue_id = ?`,
	deleteQueueParentingById:         `delete from queue_parenting where rowid = ?`,
	selectQueueById:                  `select queue from queues where rowid = ?`,
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
	selectFromNotificationQueues:       `select rowid, result_id, filename, line from notification_queues`,
	deleteFromNotificationQueues:       `delete from notification_queues where rowid = ?`,
	deleteResultByIdKey:                `delete from results where rowid = ?`,
	deleteResultDataByResultId:         `delete from result_data where result_id = ?`,
	deleteQueueDataById:                `delete from queue_data where queue_id = ?`,
	deleteQueueById:                    `delete from queues where rowid = ?`,
	deleteConnectionDataByConnectionId: `delete from connection_data where connection_id = ?`,
	connectionIdForQueue:               `select connection_id from queues where rowid = ?`,
	deleteConnectionById:               `delete from connections where rowid = ?`,
	pidIdForConnection:                 `select pid_id from connections where rowid = ?`,
	deletePidById:                      `delete from pids where rowid = ?`,
	messageIdForQueue:                  `select messageid_id from queues where rowid = ?`,
	countWithMessageIdUsageById:        `select usage_counter from messageids where rowid = ?`,
	deleteMessageId:                    `delete from messageids where rowid = ?`,
	incrementConnectionUsageById:       `update connections set usage_counter = usage_counter + 1 where rowid = ?`,
	decrementConnectionUsageById:       `update connections set usage_counter = usage_counter - 1 where rowid = ?`,
	countPidUsageById:                  `select usage_counter from pids where rowid = ?`,
	countConnectionUsageById:           `select usage_counter from connections where rowid = ?`,
	countPidUsageByPidId:               `select usage_counter from pids where rowid = ?`,
	incrementPidUsageById:              `update pids set usage_counter = usage_counter + 1 where rowid = ?`,
	decrementPidUsageById:              `update pids set usage_counter = usage_counter - 1 where rowid = ?`,
}

// TODO: close such statements when the tracker is deleted!!!
func prepareTrackerRwStmts(conn dbconn.RwConn) (trackerStmts, error) {
	stmts := trackerStmts{}

	for k, v := range trackerStmtsText {
		stmt, err := conn.Prepare(v)
		if err != nil {
			return trackerStmts{}, errorutil.Wrap(err)
		}

		stmts[k] = stmt
	}

	return stmts, nil
}
