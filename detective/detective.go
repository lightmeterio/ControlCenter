// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package detective

import (
	"context"
	"database/sql"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser"
	"gitlab.com/lightmeter/controlcenter/util/emailutil"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"strconv"
	"time"
)

const resultsPerPage = 100

type Detective interface {
	CheckMessageDelivery(context.Context, string, string, timeutil.TimeInterval, int) (*MessagesPage, error)
}

type sqlDetective struct {
	pool *dbconn.RoPool
}

func New(pool *dbconn.RoPool) (Detective, error) {
	setup := func(db *dbconn.RoPooledConn) error {
		checkMessageDelivery, err := db.Prepare(`
			select total, delivery_ts, status, dsn from (
				select row_number() over (order by delivery_ts),
				count() over () as total,
				delivery_ts, status, dsn
				from
					deliveries d
				inner join
					remote_domains sender_domain    on sender_domain.id    = d.sender_domain_part_id
				inner join
					remote_domains recipient_domain on recipient_domain.id = d.recipient_domain_part_id
				where
					sender_local_part    = ? and sender_domain.domain    = ? and
					recipient_local_part = ? and recipient_domain.domain = ? and
					delivery_ts between ? and ?
			)
			order by delivery_ts
			limit ?
			offset ?
			`)

		if err != nil {
			return errorutil.Wrap(err)
		}

		defer func() {
			if err != nil {
				errorutil.MustSucceed(checkMessageDelivery.Close(), "Closing checkMessageDelivery")
			}
		}()

		db.Stmts["checkMessageDelivery"] = checkMessageDelivery

		return nil
	}

	if err := pool.ForEach(setup); err != nil {
		return nil, errorutil.Wrap(err)
	}

	return &sqlDetective{
		pool: pool,
	}, nil
}

func (d sqlDetective) CheckMessageDelivery(ctx context.Context, mailFrom string, mailTo string, interval timeutil.TimeInterval, page int) (*MessagesPage, error) {
	conn, release := d.pool.Acquire()

	defer release()

	return checkMessageDelivery(ctx, conn.Stmts["checkMessageDelivery"], mailFrom, mailTo, interval, page)
}

type MessagesPage struct {
	PageNumber   int               `json:"page"`
	FirstPage    int               `json:"first_page"`
	LastPage     int               `json:"last_page"`
	TotalResults int               `json:"total"`
	Messages     []MessageDelivery `json:"messages"`
}

type MessageDelivery struct {
	Time   time.Time `json:"time"`
	Status string    `json:"status"`
	Dsn    string    `json:"dsn"`
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
		messages = make([]MessageDelivery, 0)
	)

	for rows.Next() {
		var (
			status string
			dsn    string
			ts     int
		)

		if err := rows.Scan(&total, &ts, &status, &dsn); err != nil {
			return nil, errorutil.Wrap(err)
		}

		intstatus, err := strconv.Atoi(status)

		if err != nil {
			return nil, errorutil.Wrap(err)
		}

		status = parser.SmtpStatus(intstatus).String()

		messages = append(messages, MessageDelivery{time.Unix(int64(ts), 0).In(time.UTC), status, dsn})
	}

	messagesPage := MessagesPage{
		PageNumber:   page,
		FirstPage:    1,
		LastPage:     total/resultsPerPage + 1,
		TotalResults: total,
		Messages:     messages,
	}

	if err := rows.Err(); err != nil {
		return nil, errorutil.Wrap(err)
	}

	return &messagesPage, nil
}
