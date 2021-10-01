// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package metadata

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/imdario/mergo"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	_ "gitlab.com/lightmeter/controlcenter/metadata/migrations"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"reflect"
)

const UuidMetaKey = "uuid"

var (
	ErrNoSuchKey = errors.New("No Such Key")
)

type Key interface{}
type Value = interface{}

type Reader interface {
	Retrieve(context.Context, Key) (Value, error)
	RetrieveJson(context.Context, Key, Value) error
}

type simpleReader struct {
	pool *dbconn.RoPool
}

func NewReader(pool *dbconn.RoPool) Reader {
	return &simpleReader{pool: pool}
}

type Writer struct {
	db dbconn.RwConn
}

type Handler struct {
	Reader Reader
	Writer *Writer
}

func NewHandler(conn *dbconn.PooledPair) (*Handler, error) {
	reader := &simpleReader{pool: conn.RoConnPool}
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
	if err := writer.db.Tx(func(tx *sql.Tx) error {
		return Store(ctx, tx, items)
	}); err != nil {
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

func Retrieve(ctx context.Context, tx *sql.Tx, key Key, value Value) error {
	if err := retrieve(ctx, tx, key, value); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

type queryiable interface {
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}

func retrieve(ctx context.Context, q queryiable, key Key, value interface{}) error {
	err := q.QueryRowContext(ctx, `select value from meta where key = ?`, key).Scan(value)

	if err == nil {
		return nil
	}

	if errors.Is(err, sql.ErrNoRows) {
		return ErrNoSuchKey
	}

	return errorutil.Wrap(err)
}

func (reader *simpleReader) Retrieve(ctx context.Context, key Key) (Value, error) {
	conn, release, err := reader.pool.AcquireContext(ctx)
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	defer release()

	var v Value

	if err := retrieve(ctx, conn, key, &v); err != nil {
		return nil, errorutil.Wrap(err)
	}

	return v, nil
}

func (writer *Writer) StoreJson(ctx context.Context, key Key, value Value) error {
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

func (reader *simpleReader) RetrieveJson(ctx context.Context, key Key, value Value) error {
	reflectValue := reflect.ValueOf(value)

	if reflectValue.Kind() != reflect.Ptr {
		panic("value isn't a pointer")
	}

	conn, release, err := reader.pool.AcquireContext(ctx)
	if err != nil {
		return errorutil.Wrap(err)
	}

	defer release()

	var v string
	if err := retrieve(ctx, conn, key, &v); err != nil {
		return errorutil.Wrap(err)
	}

	if err := json.Unmarshal([]byte(v), value); err != nil {
		return errorutil.Wrap(err, "could not Unmarshal values")
	}

	return nil
}

type DefaultValues map[string]interface{}

type defaultedReader struct {
	simpleReader  *simpleReader
	defaultValues DefaultValues
}

func (r *defaultedReader) Retrieve(ctx context.Context, key Key) (Value, error) {
	v, err := r.simpleReader.Retrieve(ctx, key)
	if err != nil && !errors.Is(err, ErrNoSuchKey) {
		return nil, errorutil.Wrap(err)
	}

	if err == nil {
		return v, nil
	}

	keyAsString, ok := key.(string)
	if !ok {
		return v, nil
	}

	defaultValue, ok := r.defaultValues[keyAsString]
	if !ok {
		return nil, errorutil.Wrap(ErrNoSuchKey)
	}

	return defaultValue, nil
}

func (r *defaultedReader) RetrieveJson(ctx context.Context, key Key, value Value) error {
	err := r.simpleReader.RetrieveJson(ctx, key, value)

	if err != nil && !errors.Is(err, ErrNoSuchKey) {
		return errorutil.Wrap(err)
	}

	keyAsString, ok := key.(string)
	if !ok {
		return nil
	}

	defaultValue, hasDefaults := r.defaultValues[keyAsString]

	if !hasDefaults {
		if err != nil && errors.Is(err, ErrNoSuchKey) {
			return errorutil.Wrap(ErrNoSuchKey)
		}

		// no need to merge
		return nil
	}

	if err := mergo.Map(value, defaultValue); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func NewDefaultedHandler(conn *dbconn.PooledPair, defaultValues DefaultValues) (*Handler, error) {
	simpleReader := &simpleReader{pool: conn.RoConnPool}
	writer := &Writer{conn.RwConn}

	return &Handler{
		Reader: &defaultedReader{simpleReader: simpleReader, defaultValues: defaultValues},
		Writer: writer,
	}, nil
}
