// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package collector

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"gitlab.com/lightmeter/controlcenter/intel/receptor"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"time"
)

type DispatchedReport struct {
	ID           int64       `json:"id"`
	CreationTime time.Time   `json:"creation_time"`
	DispatchTime time.Time   `json:"dispatch_time"`
	Kind         string      `json:"kind"`
	Value        interface{} `json:"value"`
}

type Accessor struct {
	pool *dbconn.RoPool
}

const (
	lastReportsQuery   = iota
	statusMessageQuery = iota
)

func NewAccessor(pool *dbconn.RoPool) (*Accessor, error) {
	if err := pool.ForEach(func(conn *dbconn.RoPooledConn) error {
		//nolint:sqlclosecheck
		if err := conn.PrepareStmt(`
			select id, time, dispatched_time, identifier, value
			from queued_reports
			where dispatched_time != 0
			order by dispatched_time desc
			limit 20
		`, lastReportsQuery); err != nil {
			return errorutil.Wrap(err)
		}

		//nolint:sqlclosecheck
		if err := conn.PrepareStmt(`
			select content
			from events
			where content_type = 'notification'
			order by id desc
			limit 1
		`, statusMessageQuery); err != nil {
			return errorutil.Wrap(err)
		}

		return nil
	}); err != nil {
		return nil, errorutil.Wrap(err)
	}

	return &Accessor{pool: pool}, nil
}

func (intelAccessor *Accessor) GetDispatchedReports(ctx context.Context) (reports []DispatchedReport, err error) {
	conn, release, err := intelAccessor.pool.AcquireContext(ctx)
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	defer release()

	//nolint:sqlclosecheck
	r, err := conn.GetStmt(lastReportsQuery).QueryContext(ctx)

	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	defer errorutil.DeferredClose(r, &err)

	for r.Next() {
		var (
			id           int64
			creationTime int64
			dispatchTime int64
			kind         string
			value        []byte
		)

		if err := r.Scan(&id, &creationTime, &dispatchTime, &kind, &value); err != nil {
			return nil, errorutil.Wrap(err)
		}

		var valueObj interface{}
		if err := json.Unmarshal(value, &valueObj); err != nil {
			return nil, errorutil.Wrap(err)
		}

		reports = append(reports, DispatchedReport{
			ID:           id,
			CreationTime: time.Unix(creationTime, 0).In(time.UTC),
			DispatchTime: time.Unix(dispatchTime, 0).In(time.UTC),
			Kind:         kind,
			Value:        valueObj,
		})
	}

	if err := r.Err(); err != nil {
		return nil, errorutil.Wrap(err)
	}

	return reports, nil
}

func (intelAccessor *Accessor) GetStatusMessage(ctx context.Context) (*receptor.Event, error) {
	conn, release, err := intelAccessor.pool.AcquireContext(ctx)
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	defer release()

	var (
		content string
		event   receptor.Event
	)

	//nolint:sqlclosecheck
	err = conn.GetStmt(statusMessageQuery).QueryRowContext(ctx).Scan(&content)

	if err != nil && errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}

	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	if err := json.Unmarshal([]byte(content), &event); err != nil {
		return nil, errorutil.Wrap(err)
	}

	return &event, nil
}
