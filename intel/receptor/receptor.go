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
	bruteforce.Checker
}

type Options struct {
	PollInterval time.Duration
	InstanceID   string
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
		id      string
		rawTime string
	)

	// no last event known
	err := tx.QueryRow(`select json_extract(content, '$.id'), json_extract(content, '$.creation_time') from events order by id desc limit 1`).Scan(&id, &rawTime)
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		return Payload{InstanceID: instanceID, LastKnownEventID: "", CreationTime: time.Time{}}, nil
	}

	if err != nil {
		return Payload{}, errorutil.Wrap(err)
	}

	creationTime, err := time.Parse(time.RFC3339, rawTime)
	if err != nil {
		return Payload{}, errorutil.Wrap(err)
	}

	return Payload{InstanceID: instanceID, LastKnownEventID: id, CreationTime: creationTime.In(time.UTC)}, nil
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
		Checker: &dbBruteForceChecker{pool: pool, actions: actions, listMaxSize: 100},
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

// Implements bruteforce.Checker
func (r *dbBruteForceChecker) Step(now time.Time, withResults func(bruteforce.SummaryResult) error) error {
	conn, release := r.pool.Acquire()
	defer release()

	rows, err := conn.Query(`
		select
			id, json_extract(content, "$.blocked_ips.ips")
		from
			events
		where
			content_type = "blocked_ips" and
			lm_json_time_to_timestamp(json_extract(content, '$.creation_time')) < ? and dismissing_time is null`, now.Unix())

	if err != nil {
		return errorutil.Wrap(err)
	}

	defer func() {
		// TODO: replace this when !852 is merged
		errorutil.MustSucceed(rows.Close())
	}()

	var result bruteforce.SummaryResult

	var id int64

	for rows.Next() {
		var rawContent string

		if err := rows.Scan(&id, &rawContent); err != nil {
			return errorutil.Wrap(err)
		}

		var ips []BlockedIP

		if err := json.Unmarshal([]byte(rawContent), &ips); err != nil {
			return errorutil.Wrap(err)
		}

		sort.Slice(ips, func(i, j int) bool {
			// reverse, higher counts first
			return ips[i].Count > ips[j].Count
		})

		result.TopIPs = make([]bruteforce.BlockedIP, 0, len(ips))

		for i, ip := range ips {
			if i < r.listMaxSize {
				result.TopIPs = append(result.TopIPs, bruteforce.BlockedIP{Address: ip.Address, Count: ip.Count})
			}

			result.TotalNumber += ip.Count
		}
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

	r.actions <- func(tx *sql.Tx, _ dbconn.TxPreparedStmts) error {
		if err := DismissEventByID(tx, id, now); err != nil {
			return errorutil.Wrap(err)
		}

		return nil
	}

	return nil
}

func DismissEventByID(tx *sql.Tx, id int64, time time.Time) error {
	_, err := tx.Exec(`update events set dismissing_time = ? where id = ?`, time.Unix(), id)
	return err
}
