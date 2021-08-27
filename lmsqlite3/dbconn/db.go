// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package dbconn

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/migrator"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"path"
	"sync"
)

type DB = PooledPair

var (
	workspace string
	dbNames   = []string{"auth", "intel", "intel-collector", "insights", "master"}
	dbs       = map[string]*DB{}
)

func InitialiseDatabasesWithWorkspace(workspaceDirectory string) error {
	var once sync.Once
	once.Do(func() {
		workspace = workspaceDirectory

		for _, dbName := range dbNames {
			db, err := newDb(dbName)

			if err != nil {
				log.Warn().Msgf("Failed opening database '%s' with error: %v", dbName, err)
				dbs[dbName] = nil
				continue
			}

			dbs[dbName] = db
		}
	})

	for _, db := range dbs {
		if db == nil {
			return errors.New("Databases could not be properly initialised")
		}
	}

	return nil
}

func newDb(dbName string) (*DB, error) {
	if workspace == "" {
		panic("Workspace for databases not set")
	}

	pooledPair, err := Open(path.Join(workspace, dbName), 5)

	if err != nil {
		return nil, err
	}

	if err := migrator.Run(pooledPair.RwConn.DB, dbName); err != nil {
		return nil, errorutil.Wrap(err)
	}

	return pooledPair, nil
}

func Db(dbName string) *DB {
	db, ok := dbs[dbName]

	if !ok {
		panic(fmt.Sprintf("Database '%s' hasn't been initialized", dbName))
	}

	return db
}

type Query struct {
	Query string
	Args  []interface{}
}

func (db *DB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	conn, release := db.RoConnPool.Acquire()
	defer release()

	return conn.Query(query, args...)
}

func (db *DB) QueryRow(query string, args ...interface{}) *sql.Row {
	return db.QueryRowContext(context.Background(), query, args...)
}

func (db *DB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	conn, release := db.RoConnPool.Acquire()
	defer release()

	return conn.QueryRowContext(ctx, query, args...)
}

func (db *DB) Write(query string, args ...interface{}) error {
	return db.Transaction(context.Background(), []Query{{query, args}})
}

func (db *DB) Transaction(ctx context.Context, queries []Query) error {
	tx, err := db.RwConn.BeginTx(ctx, nil)

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

type DatabasesCloser struct{}

func (c DatabasesCloser) Close() error {
	for _, db := range dbs {
		db.Closers.Close()
	}
	return nil
}
