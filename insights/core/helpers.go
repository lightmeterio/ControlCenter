// SPDX-FileCopyrightText: 2020,  Lightmeter <hello@lightmeter.io>
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0

package core

import (
	"database/sql"
	"errors"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"time"
)

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
