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
	_ "gitlab.com/lightmeter/controlcenter/meta/migrations"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"reflect"
)

const UuidMetaKey = "uuid"

var (
	ErrNoSuchKey = errors.New("No Such Key")
)

type Item struct {
	Key   interface{}
	Value interface{}
}

func Store(ctx context.Context, db *dbconn.DB, items []Item) error {
	queries := []dbconn.Query{}

	for _, i := range items {
		queries = append(queries, dbconn.Query{
			Query: `
				insert into meta(key, value) values(?, ?)
				on conflict(key) do update set value = ?;
			`,
			Args: []interface{}{i.Key, i.Value, i.Value},
		})
	}

	if err := db.Transaction(ctx, queries); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func Retrieve(ctx context.Context, db *dbconn.DB, key interface{}, value interface{}) error {
	err := db.QueryRowContext(ctx, `select value from meta where key = ?`, key).Scan(value)

	if err == nil {
		return nil
	}

	if errors.Is(err, sql.ErrNoRows) {
		return ErrNoSuchKey
	}

	return errorutil.Wrap(err)
}

func StoreJson(ctx context.Context, db *dbconn.DB, key interface{}, value interface{}) error {
	jsonBlob, err := json.Marshal(value)
	if err != nil {
		return errorutil.Wrap(err)
	}

	return Store(ctx, db, []Item{{Key: key, Value: string(jsonBlob)}})
}

func RetrieveJson(ctx context.Context, db *dbconn.DB, key interface{}, values interface{}) error {
	reflectValues := reflect.ValueOf(values)

	if reflectValues.Kind() != reflect.Ptr {
		panic("values isn't a pointer")
	}

	var v string
	if err := Retrieve(ctx, db, key, &v); err != nil {
		return errorutil.Wrap(err)
	}

	if err := json.Unmarshal([]byte(v), values); err != nil {
		return errorutil.Wrap(err, "could not Unmarshal values")
	}

	return nil
}
