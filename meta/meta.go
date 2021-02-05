// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package meta

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/migrator"
	_ "gitlab.com/lightmeter/controlcenter/meta/migrations"
	"gitlab.com/lightmeter/controlcenter/util/closeutil"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"reflect"
)

var (
	ErrNoSuchKey = errors.New("No Such Key")
)

type Reader struct {
	pool *dbconn.RoPool
}

func (reader *Reader) Close() error {
	return reader.pool.Close()
}

func (writer *Writer) Close() error {
	return writer.db.Close()
}

type Writer struct {
	db dbconn.RwConn
}

type Handler struct {
	Reader *Reader
	Writer *Writer

	closers closeutil.Closers
}

func (h *Handler) Close() error {
	if err := h.closers.Close(); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func NewHandler(conn *dbconn.PooledPair, databaseName string) (*Handler, error) {
	if err := migrator.Run(conn.RwConn.DB, databaseName); err != nil {
		return nil, errorutil.Wrap(err)
	}

	reader := &Reader{conn.RoConnPool}
	writer := &Writer{conn.RwConn}

	return &Handler{
		Reader:  reader,
		Writer:  writer,
		closers: closeutil.New(reader, writer),
	}, nil
}

type Item struct {
	Key   interface{}
	Value interface{}
}

func (writer *Writer) Store(ctx context.Context, items []Item) error {
	tx, err := writer.db.BeginTx(ctx, nil)

	if err != nil {
		return errorutil.Wrap(err)
	}

	defer func() {
		if err != nil {
			errorutil.MustSucceed(tx.Rollback())
		}
	}()

	err = Store(tx, items)

	if err != nil {
		return err
	}

	err = tx.Commit()

	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func Store(tx *sql.Tx, items []Item) error {
	for _, i := range items {
		var id int
		err := tx.QueryRow(`select rowid from meta where key = ?`, i.Key).Scan(&id)

		query, args := func() (string, []interface{}) {
			if errors.Is(err, sql.ErrNoRows) {
				return `insert into meta(key, value) values(?, ?)`, []interface{}{i.Key, i.Value}
			}

			return `update meta set value = ? where rowid = ?`, []interface{}{i.Value, id}
		}()

		if _, err := tx.Exec(query, args...); err != nil {
			return errorutil.Wrap(err)
		}
	}

	return nil
}

func retrieve(ctx context.Context, reader *Reader, key interface{}, value interface{}) error {
	conn, release := reader.pool.Acquire()

	defer release()

	err := conn.QueryRowContext(ctx, `select value from meta where key = ?`, key).Scan(value)

	if err == nil {
		return nil
	}

	if errors.Is(err, sql.ErrNoRows) {
		return ErrNoSuchKey
	}

	return errorutil.Wrap(err)
}

func (reader *Reader) Retrieve(ctx context.Context, key interface{}) (interface{}, error) {
	var v interface{}

	if err := retrieve(ctx, reader, key, &v); err != nil {
		return nil, errorutil.Wrap(err)
	}

	return v, nil
}

func (writer *Writer) StoreJson(ctx context.Context, key interface{}, value interface{}) error {
	tx, err := writer.db.BeginTx(ctx, nil)

	if err != nil {
		return errorutil.Wrap(err)
	}

	defer func() {
		if err != nil {
			errorutil.MustSucceed(tx.Rollback())
		}
	}()

	jsonBlob, err := json.Marshal(value)
	if err != nil {
		return errorutil.Wrap(err)
	}

	err = Store(tx, []Item{{Key: key, Value: string(jsonBlob)}})

	if err != nil {
		return errorutil.Wrap(err)
	}

	err = tx.Commit()

	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func (reader *Reader) RetrieveJson(ctx context.Context, key interface{}, values interface{}) error {
	reflectValues := reflect.ValueOf(values)

	if reflectValues.Kind() != reflect.Ptr {
		panic("values isn't a pointer")
	}

	var v string
	if err := retrieve(ctx, reader, key, &v); err != nil {
		return errorutil.Wrap(err)
	}

	if err := json.Unmarshal([]byte(v), values); err != nil {
		return errorutil.Wrap(err, "could not Unmarshal values")
	}

	return nil
}
