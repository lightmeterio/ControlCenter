// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package core

import (
	"context"
	"database/sql"
	"errors"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/meta"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"time"
)

type Progress struct {
	Value  *int       `json:"value,omitempty"`
	Time   *time.Time `json:"time,omitempty"`
	Active bool       `json:"active"`
}

type ProgressFetcher interface {
	// Progress, from 0 to 100
	Progress(context.Context) (Progress, error)
}

type progressFetcher struct {
	pool *dbconn.RoPool
}

func (f *progressFetcher) Progress(ctx context.Context) (Progress, error) {
	running, err := IsHistoricalImportRunningFromPool(ctx, f.pool)
	if err != nil {
		return Progress{}, errorutil.Wrap(err)
	}

	// If we skipt the import, there should be no progress info available
	skipImport, err := meta.NewReader(f.pool).Retrieve(ctx, "skip_import")
	if err != nil && !errors.Is(err, meta.ErrNoSuchKey) {
		return Progress{}, errorutil.Wrap(err)
	}

	// NOTE: Meh, SQLite converts bools into integers!
	if err == nil && skipImport.(int64) == 1 {
		// The import is done, as it's never been execute, nor will ever be (I guess :-))
		value := 100
		return Progress{Active: false, Value: &value}, nil
	}

	conn, release := f.pool.Acquire()

	defer release()

	var (
		value int
		ts    int64
	)

	err = conn.QueryRowContext(ctx, `select value, timestamp from import_progress order by rowid desc limit 1`).Scan(&value, &ts)

	if err != nil && errors.Is(err, sql.ErrNoRows) {
		// before the import starts
		return Progress{Active: running}, nil
	}

	if err != nil {
		return Progress{}, errorutil.Wrap(err)
	}

	time := time.Unix(ts, 0).In(time.UTC)

	// during or after the import process
	return Progress{Value: &value, Time: &time, Active: running}, nil
}

func NewProgressFetcher(pool *dbconn.RoPool) (ProgressFetcher, error) {
	return &progressFetcher{pool: pool}, nil
}
