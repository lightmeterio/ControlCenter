package tracking

import (
	"database/sql"
	"gitlab.com/lightmeter/controlcenter/data"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"log"
)

func dispatchAllQueues(tracker *Tracker, queuesToNotify chan<- resultInfo, tx *sql.Tx) error {
	rows, err := tx.Stmt(tracker.stmts[selectFromNotificationQueues]).Query()

	if err != nil {
		return errorutil.Wrap(err)
	}

	var (
		queueId  int64
		line     uint64 // NOTE: it's stored as an int64 in SQLite... :-(
		filename string
		id       int64
	)

	count := 0

	for rows.Next() {
		count++

		err = rows.Scan(&id, &queueId, &filename, &line)

		if err != nil {
			return errorutil.Wrap(err)
		}

		queuesToNotify <- resultInfo{id: queueId, loc: data.RecordLocation{Line: line, Filename: filename}}

		// Yes, deleting while iterating over the queues... That's supported by SQLite
		_, err = tx.Stmt(tracker.stmts[deleteFromNotificationQueues]).Exec(id)
		if err != nil {
			return errorutil.Wrap(err)
		}
	}

	err = rows.Err()
	if err != nil {
		return errorutil.Wrap(err)
	}

	if count > 0 {
		log.Println("Dispatched", count, "queues")
	}

	return nil
}
