// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package receptor

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"gitlab.com/lightmeter/controlcenter/intel/bruteforce"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/metadata"
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
	InstanceID       string
	LastKnownEventID *string
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
	Interval timeutil.TimeInterval `json:"interval"`
	Summary  bruteforce.BlockedIP  `json:"summary"`
}

const receptorContextKey = `receptor_context`

type receptorContext struct {
	CreationTime     time.Time `json:"creation_time"`
	LastKnownEventID string    `json:"last_known_event_id"`
}

func buildRequestPayload(reader metadata.Reader, instanceID string) (Payload, error) {
	var receptorContext receptorContext

	err := reader.RetrieveJson(context.Background(), receptorContextKey, &receptorContext)
	if err != nil && errors.Is(err, metadata.ErrNoSuchKey) {
		return Payload{InstanceID: instanceID}, nil
	}

	if err != nil {
		return Payload{}, errorutil.Wrap(err)
	}

	return Payload{InstanceID: instanceID, LastKnownEventID: &receptorContext.LastKnownEventID}, nil
}

func drainEvents(reader metadata.Reader, actions dbrunner.Actions, requester Requester) error {
	for {
		payload, err := buildRequestPayload(reader, options.InstanceID)
		if err != nil {
			return errorutil.Wrap(err)
		}

		response, err := requester.Request(context.Background(), payload)
		if err != nil {
			return errorutil.Wrap(err)
		}

		if response == nil {
			// on receiving "nothing", we've finished draining the server for more events
			return nil
		}

		encoded, err := json.Marshal(response)
		if err != nil {
			return errorutil.Wrap(err)
		}

		now := clock.Now()

		actions <- func(tx *sql.Tx, _ dbconn.TxPreparedStmts) error {
			const query = `insert into events(received_time, event_id, content_type, content, dismissing_time) values(?, ?, ?, ?, ?, null)`

			if _, err := tx.Exec(query, now.Unix(), response.ID, response.Type, string(encoded)); err != nil {
				return errorutil.Wrap(err)
			}

			return nil
		}
	}
}

func New(actions dbrunner.Actions, pool *dbconn.RoPool, requester Requester, options Options, clock timeutil.Clock) (*Receptor, error) {
	return &Receptor{
		Closers: closeutil.New(),
		pool:    pool,
		actions: actions,
		CancellableRunner: runner.NewCancellableRunner(func(done runner.DoneChan, cancel runner.CancelChan) {
			go func() {
				timer := time.NewTicker(options.PollInterval)

				reader := metadata.NewReader(pool)

				for {
					select {
					case <-cancel:
						timer.Stop()
						done <- nil

						return
					case <-timer.C:
						if err := drainEvents(reader, options, actions, requester); err != nil {
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
		tx.Exec(`update bruteforce_reports set used = true where id = ?`)
		return nil
	}

	return nil
}
