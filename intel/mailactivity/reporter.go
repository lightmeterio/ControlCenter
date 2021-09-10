// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package mailactivity

import (
	"database/sql"
	"gitlab.com/lightmeter/controlcenter/intel/collector"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	parser "gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser"
	"gitlab.com/lightmeter/controlcenter/tracking"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"time"
)

type report struct {
	Interval         timeutil.TimeInterval `json:"time_interval"`
	NumberOfSent     int                   `json:"sent_messages"`
	NumberOfDeferred int                   `json:"deferred_messages"`
	NumberOfBounced  int                   `json:"bounced_messages"`
	NumberOfReceived int                   `json:"received_messages"`
}

type Reporter struct {
	pool *dbconn.RoPool
}

// New receives a connection to a `deliverydb` database.
func NewReporter(pool *dbconn.RoPool) *Reporter {
	return &Reporter{pool: pool}
}

const executionInterval = 10 * time.Minute

func (r *Reporter) ExecutionInterval() time.Duration {
	return executionInterval
}

func execQuery(conn *dbconn.RoPooledConn, interval timeutil.TimeInterval, condition string, args ...interface{}) (int, error) {
	var value int

	query := `select count(*) from deliveries where delivery_ts >= ? and delivery_ts <= ? and ` + condition

	args = append([]interface{}{interval.From.Unix(), interval.To.Unix()}, args...)

	if err := conn.QueryRow(query, args...).Scan(&value); err != nil {
		return 0, errorutil.Wrap(err)
	}

	return value, nil
}

func (r *Reporter) Close() error {
	return nil
}

func (r *Reporter) Step(tx *sql.Tx, clock timeutil.Clock) error {
	conn, release := r.pool.Acquire()

	defer release()

	interval := timeutil.TimeInterval{From: clock.Now().Add(-executionInterval), To: clock.Now()}

	numberOfSent, err := execQuery(conn, interval, `direction = ? and status = ?`, int(tracking.MessageDirectionOutbound), int(parser.SentStatus))
	if err != nil {
		return errorutil.Wrap(err)
	}

	numberOfDeferred, err := execQuery(conn, interval, `direction = ? and status = ?`, int(tracking.MessageDirectionOutbound), int(parser.DeferredStatus))
	if err != nil {
		return errorutil.Wrap(err)
	}

	numberOfBounced, err := execQuery(conn, interval, `direction = ? and status = ?`, int(tracking.MessageDirectionOutbound), int(parser.BouncedStatus))
	if err != nil {
		return errorutil.Wrap(err)
	}

	numberOfReceived, err := execQuery(conn, interval, `direction = ?`, int(tracking.MessageDirectionIncoming))
	if err != nil {
		return errorutil.Wrap(err)
	}

	if numberOfSent+numberOfDeferred+numberOfBounced+numberOfReceived == 0 {
		return nil
	}

	report := report{
		Interval:         interval,
		NumberOfSent:     numberOfSent,
		NumberOfDeferred: numberOfDeferred,
		NumberOfBounced:  numberOfBounced,
		NumberOfReceived: numberOfReceived,
	}

	if err := collector.Collect(tx, clock, r.ID(), &report); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func (r *Reporter) ID() string {
	return "mail_activity"
}
