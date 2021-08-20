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

const UuidMetaKey = "uuid"

var (
	ErrNoSuchKey = errors.New("No Such Key")
	dbMaster     = dbconn.New("master.db")
)

type Reader struct {
	pool *dbconn.RoPool
}

func (reader *Reader) Close() error {
	return reader.pool.Close()
}

func NewReader(pool *dbconn.RoPool) *Reader {
	return &Reader{pool: pool}
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

	reader := NewReader(conn.RoConnPool)
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

// TODO: remove Writer
func (*Writer) Store(ctx context.Context, items []Item) error {
	return Store(ctx, nil, items)
}

// TODO remove tx
func Store(ctx context.Context, tx *sql.Tx, items []Item) error {
	queries := []dbconn.Query{}

	for _, i := range items {
		queries = append(queries, dbconn.Query{
			`
				insert into meta(key, value) values(?, ?)
				on conflict(key) do update set value = ?;
			`,
			[]interface{}{i.Key, i.Value, i.Value},
		})
	}

	if err := dbMaster.Transaction(ctx, queries); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

// TODO: remove tx
func Retrieve(ctx context.Context, tx *sql.Tx, key interface{}, value interface{}) error {
	err := dbMaster.QueryRowContext(ctx, `select value from meta where key = ?`, key).Scan(value)

	if err == nil {
		return nil
	}

	if errors.Is(err, sql.ErrNoRows) {
		return ErrNoSuchKey
	}

	return errorutil.Wrap(err)
}

// remove Reader
func (reader *Reader) Retrieve(ctx context.Context, key interface{}) (interface{}, error) {
	var v interface{}

	if err := Retrieve(ctx, nil, key, &v); err != nil {
		return nil, errorutil.Wrap(err)
	}

	return v, nil
}

// TODO: remove Writer
func (*Writer) StoreJson(ctx context.Context, key interface{}, value interface{}) error {
	jsonBlob, err := json.Marshal(value)
	if err != nil {
		return errorutil.Wrap(err)
	}

	return Store(ctx, nil, []Item{{Key: key, Value: string(jsonBlob)}})
}

// TODO: remove Reader
func (reader *Reader) RetrieveJson(ctx context.Context, key interface{}, values interface{}) error {
	reflectValues := reflect.ValueOf(values)

	if reflectValues.Kind() != reflect.Ptr {
		panic("values isn't a pointer")
	}

	var v string
	if err := Retrieve(ctx, nil, key, &v); err != nil {
		return errorutil.Wrap(err)
	}

	if err := json.Unmarshal([]byte(v), values); err != nil {
		return errorutil.Wrap(err, "could not Unmarshal values")
	}

	return nil
}
