// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package receptor

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"gitlab.com/lightmeter/controlcenter/intel/bruteforce"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/pkg/dbrunner"
	"gitlab.com/lightmeter/controlcenter/pkg/runner"
	"gitlab.com/lightmeter/controlcenter/util/closeutil"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
)

// Receptor polls the netint server and stores
// any new information in the database
type Receptor struct {
	runner.CancellableRunner
	closeutil.Closers

	pool    *dbconn.RoPool
	actions dbrunner.Actions
}

type Options struct {
	PollInterval time.Duration
	InstanceID   string
}

type Requester interface {
	Request(context.Context, Payload) (*Event, error)
}

type Payload struct {
	Time             time.Time
	InstanceID       string
	LastKnownEventID string
}

type Event struct {
	ID                  string               `json:"id"`
	Type                string               `json:"type"`
	CreationTime        time.Time            `json:"creation_time"`
	MessageNotification *MessageNotification `json:"message_notification"`
	ActionLink          *ActionLink          `json:"action_link"`
	BlockedIPs          *BlockedIPs          `json:"blocked_ips"`
}

type MessageNotification struct {
	Message string `json:"message"`
}

type ActionLink struct {
	Link  string `json:"link"`
	Label string `json:"label"`
}

type BlockedIPs struct {
	Interval timeutil.TimeInterval    `json:"interval"`
	Summary  bruteforce.SummaryResult `json:"summary"`
}

func buildRequestPayload(tx *sql.Tx, instanceID string) (Payload, error) {
	var (
		id      string
		rawTime string
	)

	// no last event known
	err := tx.QueryRow(`select json_extract(content, '$.id'), json_extract(content, '$.creation_time') from events order by id desc limit 1`).Scan(&id, &rawTime)
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		return Payload{InstanceID: instanceID, LastKnownEventID: "", Time: time.Time{}}, nil
	}

	if err != nil {
		return Payload{}, errorutil.Wrap(err)
	}

	creationTime, err := time.Parse(time.RFC3339, rawTime)
	if err != nil {
		return Payload{}, errorutil.Wrap(err)
	}

	return Payload{InstanceID: instanceID, LastKnownEventID: id, Time: creationTime.In(time.UTC)}, nil
}

func fetchNextEvent(tx *sql.Tx, options Options, requester Requester, clock timeutil.Clock) (*Event, error) {
	payload, err := buildRequestPayload(tx, options.InstanceID)
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	// FIXME: doing a request in the middle of a transaction is bad, very bad.
	// but unfortunately this is the only way for now :-(
	response, err := requester.Request(context.Background(), payload)
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
		Closers: closeutil.New(),
		pool:    pool,
		actions: actions,
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

// Implements bruteforce.Checker
func (r *Receptor) Step(now time.Time, withResults func(bruteforce.SummaryResult) error) error {
	conn, release := r.pool.Acquire()
	defer release()

	rows, err := conn.Query(`select content from bruteforce_reports where not used`)
	if err != nil {
		return errorutil.Wrap(err)
	}

	defer func() {
		// TODO: replace this when !852 is merged
		errorutil.MustSucceed(rows.Close())
	}()

	var result bruteforce.SummaryResult

	for rows.Next() {
	}

	if err := rows.Err(); err != nil {
		return errorutil.Wrap(err)
	}

	if err := withResults(result); err != nil {
		return errorutil.Wrap(err)
	}

	r.actions <- func(tx *sql.Tx, _ dbconn.TxPreparedStmts) error {
		_, _ = tx.Exec(`update bruteforce_reports set used = true where id = ?`)
		return nil
	}

	return nil
}
