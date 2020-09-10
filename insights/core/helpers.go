package core

import (
	"database/sql"
	"gitlab.com/lightmeter/controlcenter/util"
	"time"
)

func StoreLastDetectorExecution(tx *sql.Tx, kind string, time time.Time) error {
	var id int64
	var ts int64
	err := tx.QueryRow(`select rowid, ts from last_detector_execution where kind = ?`, kind).Scan(&id, &ts)

	query, args := func() (string, []interface{}) {
		if err != sql.ErrNoRows {
			return `update last_detector_execution set ts = ? where rowid = ?`, []interface{}{time.Unix(), id}
		}

		return `insert into last_detector_execution(ts, kind) values(?, ?)`, []interface{}{time.Unix(), kind}
	}()

	if _, err := tx.Exec(query, args...); err != nil {
		return util.WrapError(err)
	}

	return nil
}

func RetrieveLastDetectorExecution(tx *sql.Tx, kind string) (time.Time, error) {
	var ts int64
	err := tx.QueryRow(`select ts from last_detector_execution where kind = ?`, kind).Scan(&ts)

	if err == sql.ErrNoRows {
		return time.Time{}, nil
	}

	if err != nil {
		return time.Time{}, util.WrapError(err)
	}

	return time.Unix(ts, 0), nil
}

func SetupAuxTables(tx *sql.Tx) error {
	query := `create table if not exists last_detector_execution(ts integer, kind text)`

	if _, err := tx.Exec(query); err != nil {
		return util.WrapError(err)
	}

	return nil
}
