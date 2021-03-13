// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
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

const HistoricalImportKey = "historical_import_running"

func DisableHistoricalImportFlag(ctx context.Context, tx *sql.Tx) error {
	if err := meta.Store(ctx, tx, []meta.Item{{Key: HistoricalImportKey, Value: false}}); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func EnableHistoricalImportFlag(ctx context.Context, tx *sql.Tx) error {
	if err := meta.Store(ctx, tx, []meta.Item{{Key: HistoricalImportKey, Value: true}}); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func IsHistoricalImportRunning(ctx context.Context, tx *sql.Tx) (bool, error) {
	var running bool

	err := meta.Retrieve(ctx, tx, HistoricalImportKey, &running)

	if err != nil && errors.Is(err, meta.ErrNoSuchKey) {
		return false, nil
	}

	if err != nil {
		return false, errorutil.Wrap(err)
	}

	return running, nil
}

func IsHistoricalImportRunningFromPool(ctx context.Context, pool *dbconn.RoPool) (bool, error) {
	v, err := meta.NewReader(pool).Retrieve(ctx, HistoricalImportKey)

	if err != nil && errors.Is(err, meta.ErrNoSuchKey) {
		return false, nil
	}

	if err != nil {
		return false, errorutil.Wrap(err)
	}

	// NOTE: Meh, SQLite converts bool into int64...
	return v.(int64) != 0, nil
}

func ArchiveInsight(ctx context.Context, tx *sql.Tx, id int64, time time.Time) error {
	if _, err := tx.ExecContext(ctx, "insert into insights_status(insight_id, status, timestamp) values(?, ?, ?)", id, ArchivedCategory, time.Unix()); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func ArchiveInsightIfHistoricalImportIsRunning(ctx context.Context, tx *sql.Tx, id int64, time time.Time) error {
	running, err := IsHistoricalImportRunning(ctx, tx)
	if err != nil {
		return errorutil.Wrap(err)
	}

	if !running {
		return nil
	}

	if err := ArchiveInsight(ctx, tx, id, time); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func StoreLastDetectorExecution(tx *sql.Tx, kind string, time time.Time) error {
	var (
		id int64
		ts int64
	)

	err := tx.QueryRow(`select rowid, ts from last_detector_execution where kind = ?`, kind).Scan(&id, &ts)

	query, args := func() (string, []interface{}) {
		if !errors.Is(err, sql.ErrNoRows) {
			return `update last_detector_execution set ts = ? where rowid = ?`, []interface{}{time.Unix(), id}
		}

		return `insert into last_detector_execution(ts, kind) values(?, ?)`, []interface{}{time.Unix(), kind}
	}()

	if _, err := tx.Exec(query, args...); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func RetrieveLastDetectorExecution(tx *sql.Tx, kind string) (time.Time, error) {
	var ts int64
	err := tx.QueryRow(`select ts from last_detector_execution where kind = ?`, kind).Scan(&ts)

	if errors.Is(err, sql.ErrNoRows) {
		return time.Time{}, nil
	}

	if err != nil {
		return time.Time{}, errorutil.Wrap(err)
	}

	return time.Unix(ts, 0), nil
}
