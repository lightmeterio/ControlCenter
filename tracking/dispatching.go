package tracking

import (
	"database/sql"
	"github.com/rs/zerolog/log"
	"gitlab.com/lightmeter/controlcenter/data"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
)

func dispatchAllResults(tracker *Tracker, resultsToNotify chan<- resultInfo, tx *sql.Tx) error {
	rows, err := tx.Stmt(tracker.stmts[selectFromNotificationQueues]).Query()

	if err != nil {
		return errorutil.Wrap(err)
	}

	var (
		resultId int64
		line     uint64 // NOTE: it's stored as an int64 in SQLite... :-(
		filename string
		id       int64
	)

	count := 0

	for rows.Next() {
		count++

		err = rows.Scan(&id, &resultId, &filename, &line)

		if err != nil {
			return errorutil.Wrap(err)
		}

		resultInfo := resultInfo{id: resultId, loc: data.RecordLocation{Line: line, Filename: filename}}

		resultsToNotify <- resultInfo

		// Yes, deleting while iterating over the results... That's supported by SQLite
		_, err = tx.Stmt(tracker.stmts[deleteFromNotificationQueues]).Exec(id)
		if err != nil {
			return errorutil.Wrap(err)
		}
	}

	if err := rows.Err(); err != nil {
		return errorutil.Wrap(err)
	}

	if count > 0 {
		log.Debug().Msgf("Tracker has dispatched %v messages", count)
	}

	return nil
}
