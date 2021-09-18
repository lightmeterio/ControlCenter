// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package metadata

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	_ "gitlab.com/lightmeter/controlcenter/metadata/migrations"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"reflect"
)

const UuidMetaKey = "uuid"

var (
	ErrNoSuchKey = errors.New("No Such Key")
)

type Reader struct {
	pool *dbconn.RoPool
}

func NewReader(pool *dbconn.RoPool) *Reader {
	return &Reader{pool: pool}
}

type Writer struct {
	db dbconn.RwConn
}

type Handler struct {
	Reader *Reader
	Writer *Writer
}

func NewHandler(conn *dbconn.PooledPair) (*Handler, error) {
	reader := NewReader(conn.RoConnPool)
	writer := &Writer{conn.RwConn}

	return &Handler{
		Reader: reader,
		Writer: writer,
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

	err = Store(ctx, tx, items)

	if err != nil {
		return err
	}

	err = tx.Commit()

	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func Store(ctx context.Context, tx *sql.Tx, items []Item) error {
	for _, i := range items {
		if _, err := tx.Exec(`
			insert into meta(key, value) values(?, ?)
			on conflict(key) do update set value = ?;
		`, i.Key, i.Value, i.Value); err != nil {
			return errorutil.Wrap(err)
		}
	}

	return nil
}

func Retrieve(ctx context.Context, tx *sql.Tx, key interface{}, value interface{}) error {
	// TODO: unify this code with the one from retrieve()!!!
	err := tx.QueryRowContext(ctx, `select value from meta where key = ?`, key).Scan(value)

	if err == nil {
		return nil
	}

	if errors.Is(err, sql.ErrNoRows) {
		return ErrNoSuchKey
	}

	return errorutil.Wrap(err)
}

func retrieve(ctx context.Context, reader *Reader, key interface{}, value interface{}) error {
	conn, release, err := reader.pool.AcquireContext(ctx)
	if err != nil {
		return errorutil.Wrap(err)
	}

	defer release()

	err = conn.QueryRowContext(ctx, `select value from meta where key = ?`, key).Scan(value)
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

	err = Store(ctx, tx, []Item{{Key: key, Value: string(jsonBlob)}})

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
