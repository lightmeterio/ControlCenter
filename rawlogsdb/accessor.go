// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package rawlogsdb

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/pkg/postfix"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"io"
	"time"
)

type ContentRow struct {
	Timestamp int64  `json:"time"`
	Content   string `json:"content"`
}

type Content struct {
	Cursor  int64        `json:"cursor"`
	Content []ContentRow `json:"content"`
}

type Accessor interface {
	FetchLogsInInterval(ctx context.Context, interval timeutil.TimeInterval, pageSize int, cursor int64) (Content, error)
	FetchLogsInIntervalToWriter(context.Context, timeutil.TimeInterval, io.Writer) error
	CountLogLinesInInterval(context.Context, timeutil.TimeInterval) (int64, error)
	FetchLogLine(context.Context, time.Time, postfix.Sum) (string, error)
}

var ErrLogLineNotFound = errors.New(`Log line not found`)

type accessor struct {
	pool *dbconn.RoPool
}

func NewAccessor(pool *dbconn.RoPool) Accessor {
	return &accessor{pool: pool}
}

func (a *accessor) FetchLogsInInterval(ctx context.Context, interval timeutil.TimeInterval, pageSize int, cursor int64) (Content, error) {
	return FetchLogsInInterval(ctx, a.pool, interval, pageSize, cursor)
}

func (a *accessor) FetchLogsInIntervalToWriter(ctx context.Context, interval timeutil.TimeInterval, w io.Writer) error {
	return FetchLogsInIntervalToWriter(ctx, a.pool, interval, w)
}

func (a *accessor) CountLogLinesInInterval(ctx context.Context, interval timeutil.TimeInterval) (int64, error) {
	return CountLogLinesInInterval(ctx, a.pool, interval)
}

func (a *accessor) FetchLogLine(ctx context.Context, time time.Time, sum postfix.Sum) (string, error) {
	return FetchLogLine(ctx, a.pool, time, sum)
}

func FetchLogsInIntervalToWriter(ctx context.Context, pool *dbconn.RoPool, interval timeutil.TimeInterval, w io.Writer) error {
	conn, release, err := pool.AcquireContext(ctx)
	if err != nil {
		return errorutil.Wrap(err)
	}

	defer release()

	query := `select content from logs where time between ? and ? order by time, id asc`

	rows, err := conn.QueryContext(ctx, query, interval.From.Unix(), interval.To.Unix())

	if err != nil {
		return errorutil.Wrap(err)
	}

	defer rows.Close()

	// really, a 4KB buffer should suffice, as it's much bigger than a line size on syslog (2097 bytes)
	const bufferLen = 4096
	line := make([]byte, 0, bufferLen)

	for rows.Next() {
		if err := rows.Scan(&line); err != nil {
			return errorutil.Wrap(err)
		}

		// 10 is \n
		line = append(line, byte(10))

		if _, err := io.Copy(w, bytes.NewReader(line)); err != nil {
			return errorutil.Wrap(err)
		}
	}

	if err := rows.Err(); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func CountLogLinesInInterval(ctx context.Context, pool *dbconn.RoPool, interval timeutil.TimeInterval) (int64, error) {
	conn, release, err := pool.AcquireContext(ctx)
	if err != nil {
		return 0, errorutil.Wrap(err)
	}

	defer release()

	var count int64

	query := `select count(*) from logs where time between ? and ?`

	if err := conn.QueryRowContext(ctx, query, interval.From.Unix(), interval.To.Unix()).Scan(&count); err != nil {
		return 0, errorutil.Wrap(err)
	}

	return count, nil
}

func FetchLogsInInterval(ctx context.Context, pool *dbconn.RoPool, interval timeutil.TimeInterval, pageSize int, cursor int64) (Content, error) {
	conn, release, err := pool.AcquireContext(ctx)
	if err != nil {
		return Content{}, errorutil.Wrap(err)
	}

	defer release()

	rowsContent := make([]ContentRow, 0, pageSize)

	query := `select id, time, content from logs where time between ? and ? and id > ? order by time, id asc limit ?`

	rows, err := conn.QueryContext(ctx, query, interval.From.Unix(), interval.To.Unix(), cursor, pageSize)
	if err != nil {
		return Content{}, errorutil.Wrap(err)
	}

	defer rows.Close()

	var nextCursor int64

	for rows.Next() {
		row := ContentRow{}

		if err := rows.Scan(&nextCursor, &row.Timestamp, &row.Content); err != nil {
			return Content{}, errorutil.Wrap(err)
		}

		rowsContent = append(rowsContent, row)
	}

	if err := rows.Err(); err != nil {
		return Content{}, errorutil.Wrap(err)
	}

	return Content{
		Cursor:  nextCursor,
		Content: rowsContent,
	}, nil
}

func MostRecentLogTimeAndSum(ctx context.Context, pool *dbconn.RoPool) (postfix.SumPair, error) {
	conn, release, err := pool.AcquireContext(ctx)
	if err != nil {
		return postfix.SumPair{}, errorutil.Wrap(err)
	}

	defer release()

	var (
		ts  int64
		sum int64
	)

	err = conn.QueryRow(`select time, checksum from logs order by id desc limit 1`).Scan(&ts, &sum)

	if err != nil && errors.Is(err, sql.ErrNoRows) {
		return postfix.SumPair{}, nil
	}

	if err != nil {
		return postfix.SumPair{}, errorutil.Wrap(err)
	}

	return postfix.SumPair{Time: time.Unix(ts, 0).In(time.UTC), Sum: (*postfix.Sum)(&sum)}, nil
}

func FetchLogLine(ctx context.Context, pool *dbconn.RoPool, t time.Time, sum postfix.Sum) (string, error) {
	var line string

	conn, release, err := pool.AcquireContext(ctx)
	if err != nil {
		return "", errorutil.Wrap(err)
	}

	defer release()

	err = conn.QueryRow(`select content from logs where time = ? and checksum = ?`, t.Unix(), sum).Scan(&line)
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		return "", ErrLogLineNotFound
	}

	if err != nil {
		return "", errorutil.Wrap(err)
	}

	return line, nil
}
