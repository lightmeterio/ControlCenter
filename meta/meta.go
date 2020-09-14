package meta

import (
	"database/sql"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/util"
)

type MetadataHandler struct {
	conn dbconn.ConnPair
}

func NewMetaDataHandler(conn dbconn.ConnPair) (*MetadataHandler, error) {
	if err := ensureMetaTableExists(conn.RwConn); err != nil {
		return nil, util.WrapError(err)
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
		return Result{}, util.WrapError(err)
	}

	defer func() {
		if err != nil {
			util.MustSucceed(tx.Rollback(), "")
		}
	}()

	r, err := Store(tx, items)

	if err != nil {
		return Result{}, err
	}

	err = tx.Commit()

	if err != nil {
		return Result{}, util.WrapError(err)
	}

	return r, nil
}

func Store(tx *sql.Tx, items []Item) (Result, error) {
	stmt, err := tx.Prepare(`insert into meta(key, value) values(?, ?)`)

	if err != nil {
		return Result{}, util.WrapError(err)
	}

	defer func() { util.MustSucceed(stmt.Close(), "") }()

	for _, i := range items {
		_, err := stmt.Exec(i.Key, i.Value)

		if err != nil {
			return Result{}, util.WrapError(err)
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
		return []interface{}{}, util.WrapError(err)
	}

	defer func() { util.MustSucceed(rows.Close(), "") }()

	results := []interface{}{}

	for rows.Next() {
		var v interface{}
		err = rows.Scan(&v)

		if err != nil {
			return []interface{}{}, util.WrapError(err)
		}

		results = append(results, v)
	}

	err = rows.Err()

	if err != nil {
		return []interface{}{}, util.WrapError(err)
	}

	return results, nil
}

func ensureMetaTableExists(conn dbconn.RwConn) error {
	if _, err := conn.Exec(`create table if not exists meta(
		key string,
		value blob
	)`); err != nil {
		return util.WrapError(err)
	}

	return nil
}
