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
	selectConnectionForPid
	insertQueueForConnection
	insertQueueData
	selectMessageIdForMessage
	insertMessageId
	updateQueueWithMessageId
	selectQueueIdForQueue
	insertQueueDataFourRows
	insertQueueParenting
	insertNotificationQueue
	selectNewQueueFromParenting
	selectQueueById
	insertResultData15Rows
	insertResultData3Rows
	insertResult
	deleteFromNotificationQueues

	selectFromNotificationQueues

	lastTrackerStmtKey
)

var trackerStmtsText = map[trackerStmtKey]string{
	insertPidOnConnection:        `insert into pids(pid, host) values(?, ?)`,
	insertConnectionOnConnection: `insert into connections(pid_id) values(?)`,
	insertConnectionDataTwoRows:  `insert into connection_data(connection_id, key, value) values(?, ?, ?), (?, ?, ?)`,
	insertConnectionData:         `insert into connection_data(connection_id, key, value) values(?, ?, ?)`,
	selectConnectionForPid: `select
		connections.rowid
	from
		connections join pids
	where
		connections.pid_id = pids.rowid 
		and pids.host = ? and pids.pid = ?`,
	insertQueueForConnection:  `insert into queues(connection_id, queue) values(?, ?)`,
	insertQueueData:           `insert into queue_data(queue_id, key, value) values(?, ?, ?)`,
	selectMessageIdForMessage: `select rowid from messageids where value = ?`,
	insertMessageId:           `insert into messageids(value) values(?)`,
	updateQueueWithMessageId:  `update queues set messageid_id = ? where queues.rowid = ?`,
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
	insertQueueParenting:        `insert into queue_parenting(orig_queue_id, new_queue_id, parenting_type) values(?, ?, ?)`,
	insertNotificationQueue:     `insert into notification_queues(queue_id, filename, line) values(?, ?, ?)`,
	selectNewQueueFromParenting: `select new_queue_id from queue_parenting where orig_queue_id = ?`,
	selectQueueById:             `select queue from queues where rowid = ?`,
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
	insertResult:                 `insert into results(queue_id) values(?)`,
	selectFromNotificationQueues: `select rowid, queue_id, filename, line from notification_queues`,
	deleteFromNotificationQueues: `delete from notification_queues where rowid = ?`,
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
