package meta

import (
	"database/sql"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/migrator"
	_ "gitlab.com/lightmeter/controlcenter/meta/migrations"

	"gitlab.com/lightmeter/controlcenter/util/errorutil"
)

type MetadataHandler struct {
	conn dbconn.ConnPair
}

func NewMetaDataHandler(conn dbconn.ConnPair, databaseName string) (*MetadataHandler, error) {

	if err := migrator.Run(conn.RwConn.DB, databaseName); err != nil {
		return nil, errorutil.Wrap(err)
	}

	return &MetadataHandler{conn}, nil
}

func (h *MetadataHandler) Close() error {
	return nil
}

type Item struct {
	Key   string
	Value interface{}
}

type Result struct {
}

func (h *MetadataHandler) Store(items []Item) (Result, error) {
	tx, err := h.conn.RwConn.Begin()

	if err != nil {
		return Result{}, errorutil.Wrap(err)
	}

	defer func() {
		if err != nil {
			errorutil.MustSucceed(tx.Rollback(), "")
		}
	}()

	r, err := Store(tx, items)

	if err != nil {
		return Result{}, err
	}

	err = tx.Commit()

	if err != nil {
		return Result{}, errorutil.Wrap(err)
	}

	return r, nil
}

func Store(tx *sql.Tx, items []Item) (Result, error) {
	stmt, err := tx.Prepare(`insert into meta(key, value) values(?, ?)`)

	if err != nil {
		return Result{}, errorutil.Wrap(err)
	}

	defer func() { errorutil.MustSucceed(stmt.Close(), "") }()

	for _, i := range items {
		_, err := stmt.Exec(i.Key, i.Value)

		if err != nil {
			return Result{}, errorutil.Wrap(err)
		}
	}

	return Result{}, nil
}

// NOTE: For some reason, rowserrcheck is not able to see that q.Err() is being called,
// so we disable the check here until the linter is fixed or someone finds the bug in this
// code.
//nolint:rowserrcheck
func (h *MetadataHandler) Retrieve(key string) ([]interface{}, error) {
	rows, err := h.conn.RoConn.Query(`select value from meta where key = ?`, key)

	if err != nil {
		return []interface{}{}, errorutil.Wrap(err)
	}

	defer func() { errorutil.MustSucceed(rows.Close(), "") }()

	results := []interface{}{}

	for rows.Next() {
		var v interface{}
		err = rows.Scan(&v)

		if err != nil {
			return []interface{}{}, errorutil.Wrap(err)
		}

		results = append(results, v)
	}

	err = rows.Err()

	if err != nil {
		return []interface{}{}, errorutil.Wrap(err)
	}

	return results, nil
}

