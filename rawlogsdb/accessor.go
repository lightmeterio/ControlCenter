// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package rawlogsdb

import (
	"context"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"io"
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
}

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
	return FetchLogsInIntervalToWriter(ctx, interval, w)
}

func FetchLogsInIntervalToWriter(ctx context.Context, internal timeutil.TimeInterval, w io.Writer) error {
	return nil
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
