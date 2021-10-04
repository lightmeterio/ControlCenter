// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package detective

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/rs/zerolog/log"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser"
	"gitlab.com/lightmeter/controlcenter/util/emailutil"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"time"
)

const resultsPerPage = 100

type Detective interface {
	CheckMessageDelivery(ctx context.Context, from, to string, interval timeutil.TimeInterval, page int) (*MessagesPage, error)
	OldestAvailableTime(context.Context) (time.Time, error)
}

type sqlDetective struct {
	pool *dbconn.RoPool
}

const (
	checkMessageDeliveryKey = iota
	oldestAvailableTimeKey
)

func New(pool *dbconn.RoPool) (Detective, error) {
	setup := func(db *dbconn.RoPooledConn) error {
		if err := db.PrepareStmt(`
			with
			sent_deliveries_filtered_by_condition(id, delivery_ts, status, dsn, queue_id, returned) as (
				select d.id, d.delivery_ts, d.status, d.dsn, dq.queue_id, false
				from
					deliveries d
				join
					remote_domains sender_domain    on sender_domain.id    = d.sender_domain_part_id
				join
					remote_domains recipient_domain on recipient_domain.id = d.recipient_domain_part_id
				join
					delivery_queue dq on dq.delivery_id = d.id
				where
					sender_local_part    = ? collate nocase and sender_domain.domain    = ? collate nocase
					and recipient_local_part = ? collate nocase and recipient_domain.domain = ? collate nocase
					and delivery_ts between ? and ?
			),
			returned_deliveries(id, delivery_ts, status, dsn, queue_id, returned) as (
				select d.id, d.delivery_ts, d.status, d.dsn, sd.queue_id, true
				from
					deliveries d
				join
					delivery_queue on delivery_queue.delivery_id = d.id
				join
					queue_parenting on delivery_queue.queue_id = queue_parenting.child_queue_id
				join
					queues qp on queue_parenting.parent_queue_id = qp.id
				join
					queues qc on queue_parenting.child_queue_id = qc.id
				join
					sent_deliveries_filtered_by_condition sd on qp.id = sd.queue_id
			),
			deliveries_filtered_by_condition(id, delivery_ts, status, dsn, queue_id, returned) as (
				select id, delivery_ts, status, dsn, queue_id, returned from sent_deliveries_filtered_by_condition
				union
				select id, delivery_ts, status, dsn, queue_id, returned from returned_deliveries
			),
			queues_filtered_by_condition(queue_id, expired_ts) as (
				select distinct delivery_queue.queue_id, expired_ts
				from deliveries_filtered_by_condition
				left join expired_queues eq on eq.queue_id = deliveries_filtered_by_condition.queue_id
				join delivery_queue on delivery_queue.delivery_id = deliveries_filtered_by_condition.id
			),
			grouped_and_computed(rn, total, delivery_ts, status, dsn, queue_id, queue, expired_ts, number_of_attempts, min_ts, max_ts, returned) as (
				select
					row_number() over (order by delivery_ts),
					count() over () as total,
					delivery_ts, status, dsn, d.queue_id, queues.name as queue, expired_ts,
					count(*) as number_of_attempts, min(delivery_ts) as min_ts, max(delivery_ts) as max_ts,
					d.returned as returned
				from deliveries_filtered_by_condition d
				join queues on d.queue_id = queues.id
				join queues_filtered_by_condition q where q.queue_id = d.queue_id
				group by d.queue_id, status, dsn
			)
			select total, status, dsn, queue, expired_ts, number_of_attempts, min_ts, max_ts, returned
			from grouped_and_computed
			order by delivery_ts, returned
			limit ?
			offset ?
			`, checkMessageDeliveryKey); err != nil {
			return errorutil.Wrap(err)
		}

		if err := db.PrepareStmt(`
			with first_delivery_queue(delivery_id) as
			(
				select delivery_id from delivery_queue order by id asc limit 1
			)
			select
				deliveries.delivery_ts
			from
				deliveries join first_delivery_queue on first_delivery_queue.delivery_id = deliveries.id
		`, oldestAvailableTimeKey); err != nil {
			return errorutil.Wrap(err)
		}

		return nil
	}

	if err := pool.ForEach(setup); err != nil {
		return nil, errorutil.Wrap(err)
	}

	return &sqlDetective{
		pool: pool,
	}, nil
}

var ErrNoAvailableLogs = errors.New(`No available logs`)

func (d *sqlDetective) CheckMessageDelivery(ctx context.Context, mailFrom string, mailTo string, interval timeutil.TimeInterval, page int) (*MessagesPage, error) {
	conn, release, err := d.pool.AcquireContext(ctx)
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	defer release()

	//nolint:sqlclosecheck
	return checkMessageDelivery(ctx, conn.GetStmt(checkMessageDeliveryKey), mailFrom, mailTo, interval, page)
}

func (d *sqlDetective) OldestAvailableTime(ctx context.Context) (time.Time, error) {
	conn, release, err := d.pool.AcquireContext(ctx)
	if err != nil {
		return time.Time{}, errorutil.Wrap(err)
	}

	defer release()

	var ts int64

	//nolint:sqlclosecheck
	err = conn.GetStmt(oldestAvailableTimeKey).QueryRowContext(ctx).Scan(&ts)

	// no available logs yet. That's fine
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		return time.Time{}, ErrNoAvailableLogs
	}

	if err != nil {
		return time.Time{}, errorutil.Wrap(err)
	}

	return time.Unix(ts, 0).In(time.UTC), nil
}

type QueueName = string

type Message struct {
	Queue   QueueName         `json:"queue"`
	Entries []MessageDelivery `json:"entries"`
}

type Messages = []Message

type MessagesPage struct {
	PageNumber   int      `json:"page"`
	FirstPage    int      `json:"first_page"`
	LastPage     int      `json:"last_page"`
	TotalResults int      `json:"total"`
	Messages     Messages `json:"messages"`
}

type Status parser.SmtpStatus

func (s Status) MarshalJSON() ([]byte, error) {
	return json.Marshal(parser.SmtpStatus(s).String())
}

func (s *Status) UnmarshalJSON(d []byte) error {
	var v string
	if err := json.Unmarshal(d, &v); err != nil {
		return errorutil.Wrap(err)
	}

	status, err := parser.ParseStatus([]byte(v))
	if err != nil {
		return errorutil.Wrap(err)
	}

	*s = Status(status)

	return nil
}

type MessageDelivery struct {
	NumberOfAttempts int        `json:"number_of_attempts"`
	TimeMin          time.Time  `json:"time_min"`
	TimeMax          time.Time  `json:"time_max"`
	Status           Status     `json:"status"`
	Dsn              string     `json:"dsn"`
	Expired          *time.Time `json:"expired"`
}

// NOTE: we are checking rows.Err(), but the linter won't see that
//nolint:rowserrcheck
func checkMessageDelivery(ctx context.Context, stmt *sql.Stmt, mailFrom string, mailTo string, interval timeutil.TimeInterval, page int) (*MessagesPage, error) {
	senderLocal, senderDomain, err := emailutil.Split(mailFrom)

	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	recipientLocal, recipientDomain, err := emailutil.Split(mailTo)

	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	queryStart := time.Now()

	defer func() {
		log.Debug().Msgf("Time to execute checkMessageDelivery: %v", time.Since(queryStart))
	}()

	rows, err := stmt.QueryContext(ctx,
		senderLocal, senderDomain,
		recipientLocal, recipientDomain,
		interval.From.Unix(), interval.To.Unix(),
		resultsPerPage, (page-1)*resultsPerPage,
	)

	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	defer func() { errorutil.MustSucceed(rows.Close()) }()

	var (
		total    int
		grouped  = 0
		messages = Messages{}
	)

	for rows.Next() {
		var (
			status           parser.SmtpStatus
			dsn              string
			queueName        QueueName
			expiredTs        *int64
			expiredTime      *time.Time
			numberOfAttempts int
			tsMin            int64
			tsMax            int64
			returned         bool
		)

		if err := rows.Scan(&total, &status, &dsn, &queueName, &expiredTs, &numberOfAttempts, &tsMin, &tsMax, &returned); err != nil {
			return nil, errorutil.Wrap(err)
		}

		if returned {
			status = parser.ReturnedStatus
		}

		index := func() int {
			for i, v := range messages {
				if v.Queue == queueName {
					grouped++
					return i
				}
			}

			messages = append(messages, Message{Queue: queueName, Entries: []MessageDelivery{}})

			return len(messages) - 1
		}()

		if expiredTs != nil {
			eT := time.Unix(*expiredTs, 0).In(time.UTC)
			expiredTime = &eT
		}

		delivery := MessageDelivery{
			NumberOfAttempts: numberOfAttempts,
			TimeMin:          time.Unix(tsMin, 0).In(time.UTC),
			TimeMax:          time.Unix(tsMax, 0).In(time.UTC),
			Status:           Status(status),
			Dsn:              dsn,
			Expired:          expiredTime,
		}

		messages[index].Entries = append(messages[index].Entries, delivery)
	}

	messagesPage := MessagesPage{
		PageNumber:   page,
		FirstPage:    1,
		LastPage:     total/resultsPerPage + 1,
		TotalResults: total - grouped,
		Messages:     messages,
	}

	if err := rows.Err(); err != nil {
		return nil, errorutil.Wrap(err)
	}

	return &messagesPage, nil
}
