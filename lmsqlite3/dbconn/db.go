// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package dbconn

import (
	"context"
	"database/sql"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"path"
)

var workspace string

func SetWorkspace(workspaceDirectory string) {
	workspace = workspaceDirectory
}

type DB struct {
	dbName     string
	pooledPair *PooledPair
}

var pooledPairs = map[string]*PooledPair{}

func (db *DB) open() error {
	if workspace == "" {
		panic("Workspace for databases not set")
	}

	if db.pooledPair != nil {
		return nil
	}

	_, found := pooledPairs[db.dbName]

	if !found {
		// TODO mutex'ed database opening
		pooledPair, err := Open(path.Join(workspace, db.dbName), 5)

		if err != nil {
			return err
		}

		pooledPairs[db.dbName] = pooledPair
	}

	db.pooledPair = pooledPairs[db.dbName]

	return nil
}

func New(dbName string) *DB {
	return &DB{dbName: dbName}
}

type Query struct {
	Query string
	Args  []interface{}
}

type Row struct {
	otherError error // non-nil if there was an error before .Scan was called (e.g. opening db)
	row        *sql.Row
}

func (db *DB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *Row {
	err := db.open()

	if err != nil {
		return &Row{otherError: errorutil.Wrap(err)}
	}

	conn, release := db.pooledPair.RoConnPool.Acquire()
	defer release()

	return &Row{row: conn.QueryRowContext(ctx, query, args...)}
}

func (r *Row) Scan(values ...interface{}) error {
	if r.otherError != nil {
		return r.otherError
	}

	return r.row.Scan(values...)
}

func (db *DB) Transaction(ctx context.Context, queries []Query) error {
	err := db.open()

	if err != nil {
		return err
	}

	tx, err := db.pooledPair.RwConn.BeginTx(ctx, nil)

	if err != nil {
		return errorutil.Wrap(err)
	}

	defer func() {
		if err != nil {
			errorutil.MustSucceed(tx.Rollback())
		}
	}()

	for _, q := range queries {
		if _, err := tx.Exec(q.Query, q.Args...); err != nil {
			return errorutil.Wrap(err)
		}
	}

	err = tx.Commit()

	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

// TODO: call CloseAll() upon lmcc termination
func CloseAll() {
	for _, db := range pooledPairs {
		db.Closers.Close()
	}
}
