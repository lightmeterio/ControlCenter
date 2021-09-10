// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package collector

import (
	"context"
	"encoding/json"
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
	lastReportsQuery = iota
)

func NewAccessor(pool *dbconn.RoPool) (*Accessor, error) {
	if err := pool.ForEach(func(conn *dbconn.RoPooledConn) error {
		//nolint:sqlclosecheck
		sql, err := conn.Prepare(`
			select id, time, dispatched_time, identifier, value
			from queued_reports
			where dispatched_time != 0
			order by dispatched_time desc
			limit 20
		`)

		if err != nil {
			return errorutil.Wrap(err)
		}

		conn.Stmts[lastReportsQuery] = sql

		return nil
	}); err != nil {
		return nil, errorutil.Wrap(err)
	}

	return &Accessor{pool: pool}, nil
}

func (intelAccessor *Accessor) GetDispatchedReports(ctx context.Context) ([]DispatchedReport, error) {
	conn, release := intelAccessor.pool.Acquire()
	defer release()

	r, err := conn.Stmts[lastReportsQuery].QueryContext(ctx)

	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	defer func() {
		errorutil.MustSucceed(r.Close())
	}()

	reports := []DispatchedReport{}

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

	return reports, nil
}
