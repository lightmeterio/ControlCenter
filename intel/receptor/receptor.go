// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package receptor

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"sort"
	"time"

	"gitlab.com/lightmeter/controlcenter/intel/blockedips"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/pkg/closers"
	"gitlab.com/lightmeter/controlcenter/pkg/dbrunner"
	"gitlab.com/lightmeter/controlcenter/pkg/runner"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
)

// Receptor polls the netint server and stores
// any new information in the database
type Receptor struct {
	runner.CancellableRunner
	closers.Closers
	blockedips.Checker
}

type Options struct {
	PollInterval              time.Duration
	InstanceID                string
	BruteForceInsightListSize int
}

type Requester interface {
	Request(context.Context, Payload) (*Event, error)
}

type Payload struct {
	CreationTime     time.Time
	InstanceID       string
	LastKnownEventID string
}

type Event struct {
	ID                  string               `json:"id"`
	Type                string               `json:"type"`
	CreationTime        time.Time            `json:"creation_time"`
	MessageNotification *MessageNotification `json:"notification"`
	BlockedIPs          *BlockedIPs          `json:"blocked_ips"`
}

type MessageNotification struct {
	Severity   string      `json:"severity"` // primary|secondary|success|danger|warning|info|light|dark (bootstrap)
	Title      string      `json:"title"`
	Message    string      `json:"message"`
	ActionLink *ActionLink `json:"action"`
}

type ActionLink struct {
	Link  string `json:"link"`
	Label string `json:"label"`
}

type BlockedIP struct {
	Address string `json:"address"`
	Count   int    `json:"count"`
}

type BlockedIPs struct {
	Interval timeutil.TimeInterval `json:"interval"`
	List     []BlockedIP           `json:"ips"`
}

func buildRequestPayload(tx *sql.Tx, instanceID string) (Payload, error) {
	var (
		id string
		ts int64
	)

	// no last event known
	err := tx.QueryRow(`select json_extract(content, '$.id'), lm_json_time_to_timestamp(json_extract(content, '$.creation_time')) from events order by id desc limit 1`).Scan(&id, &ts)
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		return Payload{InstanceID: instanceID, LastKnownEventID: "", CreationTime: time.Time{}}, nil
	}

	if err != nil {
		return Payload{}, errorutil.Wrap(err)
	}

	return Payload{InstanceID: instanceID, LastKnownEventID: id, CreationTime: time.Unix(ts, 0).In(time.UTC)}, nil
}

func fetchNextEvent(tx *sql.Tx, options Options, requester Requester, clock timeutil.Clock) (*Event, error) {
	payload, err := buildRequestPayload(tx, options.InstanceID)
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	// FIXME: doing a request in the middle of a transaction is bad, very bad.
	// but unfortunately this is the only way for now :-(
	response, err := requester.Request(context.Background(), payload)
	if err != nil && errors.Is(err, ErrRequestFailed) {
		return nil, nil
	}

	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	return response, nil
}

func eventAction(tx *sql.Tx, options Options, requester Requester, clock timeutil.Clock) error {
	event, err := fetchNextEvent(tx, options, requester, clock)
	if err != nil {
		return errorutil.Wrap(err)
	}

	if event == nil {
		return nil
	}

	encoded, err := json.Marshal(event)
	if err != nil {
		return errorutil.Wrap(err)
	}

	now := clock.Now()

	const query = `insert into events(received_time, event_id, content_type, content, dismissing_time) values(?, ?, ?, ?, null)`

	if _, err := tx.Exec(query, now.Unix(), event.ID, event.Type, string(encoded)); err != nil {
		return errorutil.Wrap(err)
	}

	if err := eventAction(tx, options, requester, clock); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func DrainEvents(actions dbrunner.Actions, options Options, requester Requester, clock timeutil.Clock) error {
	actions <- func(tx *sql.Tx, _ dbconn.TxPreparedStmts) error {
		if err := eventAction(tx, options, requester, clock); err != nil {
			return errorutil.Wrap(err)
		}

		return nil
	}

	return nil
}

func New(actions dbrunner.Actions, pool *dbconn.RoPool, requester Requester, options Options, clock timeutil.Clock) (*Receptor, error) {
	return &Receptor{
		Closers: closers.New(),
		Checker: &dbBruteForceChecker{pool: pool, actions: actions, listMaxSize: options.BruteForceInsightListSize},
		CancellableRunner: runner.NewCancellableRunner(func(done runner.DoneChan, cancel runner.CancelChan) {
			go func() {
				timer := time.NewTicker(options.PollInterval)

				for {
					select {
					case <-cancel:
						timer.Stop()
						done <- nil

						return
					case <-timer.C:
						if err := DrainEvents(actions, options, requester, clock); err != nil {
							done <- errorutil.Wrap(err)
							return
						}
					}
				}
			}()
		}),
	}, nil
}

type dbBruteForceChecker struct {
	pool        *dbconn.RoPool
	actions     dbrunner.Actions
	listMaxSize int
}

// Implements blockedips.Checker
func (r *dbBruteForceChecker) Step(interval timeutil.TimeInterval, withResults func(blockedips.SummaryResult) error) (err error) {
	conn, release := r.pool.Acquire()
	defer release()

	//nolint:sqlclosecheck
	rows, err := conn.Query(`
		select
			id, json_extract(content, "$.blocked_ips.ips"),
			lm_json_time_to_timestamp(json_extract(content, '$.blocked_ips.interval.from')),
			lm_json_time_to_timestamp(json_extract(content, '$.blocked_ips.interval.to'))
		from
			events
		where
			content_type = "blocked_ips" and
			lm_json_time_to_timestamp(json_extract(content, '$.creation_time')) between ? and ? and dismissing_time is null`, interval.From.Unix(), interval.To.Unix())

	if err != nil {
		return errorutil.Wrap(err)
	}

	defer errorutil.UpdateErrorFromCloser(rows, &err)

	var result blockedips.SummaryResult

	var ids []int64

	for rows.Next() {
		var (
			rawContent   string
			intervalFrom int64
			intervalTo   int64
			id           int64
		)

		if err := rows.Scan(&id, &rawContent, &intervalFrom, &intervalTo); err != nil {
			return errorutil.Wrap(err)
		}

		ids = append(ids, id)

		var ips []BlockedIP

		if err := json.Unmarshal([]byte(rawContent), &ips); err != nil {
			return errorutil.Wrap(err)
		}

		sort.Slice(ips, func(i, j int) bool {
			// reverse, higher counts first
			return ips[i].Count > ips[j].Count
		})

		result.TopIPs = make([]blockedips.BlockedIP, 0, len(ips))

		for i, ip := range ips {
			if i < r.listMaxSize {
				result.TopIPs = append(result.TopIPs, blockedips.BlockedIP{Address: ip.Address, Count: ip.Count})
			}

			result.TotalIPs++
			result.TotalNumber += ip.Count
		}

		result.Interval.From = time.Unix(intervalFrom, 0).In(time.UTC)
		result.Interval.To = time.Unix(intervalTo, 0).In(time.UTC)
	}

	if err := rows.Err(); err != nil {
		return errorutil.Wrap(err)
	}

	if result.TotalNumber == 0 {
		return nil
	}

	if err := withResults(result); err != nil {
		return errorutil.Wrap(err)
	}

	// NOTE: the dismissal happens asynchronously, so the next execution of Step()
	// won't see the current result event being dismissed if the interval between executions
	// is very short (like a couple of seconds)
	r.actions <- func(tx *sql.Tx, _ dbconn.TxPreparedStmts) error {
		for _, id := range ids {
			if err := DismissEventByID(tx, id, interval.To); err != nil {
				return errorutil.Wrap(err)
			}
		}

		return nil
	}

	return nil
}

func DismissEventByID(tx *sql.Tx, id int64, time time.Time) error {
	_, err := tx.Exec(`update events set dismissing_time = ? where id = ?`, time.Unix(), id)
	return err
}
