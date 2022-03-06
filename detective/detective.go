// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package detective

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/pkg/postfix"
	"gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser"
	"gitlab.com/lightmeter/controlcenter/rawlogsdb"
	"gitlab.com/lightmeter/controlcenter/tracking"
	"gitlab.com/lightmeter/controlcenter/util/emailutil"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"time"
)

const resultsPerPage = 100

type Detective interface {
	CheckMessageDelivery(ctx context.Context, from, to string, interval timeutil.TimeInterval, status int, someID string, page int) (*MessagesPage, error)
	OldestAvailableTime(context.Context) (time.Time, error)
}

type sqlDetective struct {
	deliveriesConnPool *dbconn.RoPool
	rawLogsAccessor    rawlogsdb.Accessor
}

const (
	checkMessageDeliveryKey = iota
	oldestAvailableTimeKey
)

func New(deliveriesConnPool *dbconn.RoPool, rawLogsAccessor rawlogsdb.Accessor) (Detective, error) {
	setup := func(db *dbconn.RoPooledConn) error {
		if err := db.PrepareStmt(`
			with
			sent_deliveries_filtered_by_condition(id, delivery_ts, status, dsn, queue_id, message_id, direction, returned, mailfrom, mailto, relay_id) as (
				select
					d.id, d.delivery_ts, d.status, d.dsn, dq.queue_id, mid.value, d.direction, false,
					sender_local_part    || '@' || sender_domain.domain    as mailfrom,
					recipient_local_part || '@' || recipient_domain.domain as mailto,
					d.next_relay_id
				from
					deliveries d
				join
					remote_domains sender_domain    on sender_domain.id    = d.sender_domain_part_id
				join
					remote_domains recipient_domain on recipient_domain.id = d.recipient_domain_part_id
				left join
					next_relays relay on relay.id = d.next_relay_id
				join
					delivery_queue dq on dq.delivery_id = d.id
				join
					queues q on q.id = dq.queue_id
				join
					messageids mid on mid.id = d.message_id
				where
					(sender_local_part       = ? collate nocase or ? = '') and
					(sender_domain.domain    = ? collate nocase or ? = '') and
					(recipient_local_part    = ? collate nocase or ? = '') and
					(recipient_domain.domain = ? collate nocase or relay.hostname like ? collate nocase or ? = '') and
					(delivery_ts between ? and ?) and
					(
						status = ? and status != 42 and direction = 0  -- sent emails
						or ? = 42 and direction = 1                    -- received emails
						or ? = -1
						or ? = 3 and exists(select * from expired_queues where queue_id = q.id)
					) and
					(q.name = ? or mid.value = ? or ? = '')
			),
			returned_deliveries(id, delivery_ts, status, dsn, queue_id, message_id, direction, returned, mailfrom, mailto, relay_id) as (
				select d.id, d.delivery_ts, d.status, d.dsn, sd.queue_id, mid.value, d.direction, true, mailfrom, mailto, d.next_relay_id
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
				join
					messageids mid on mid.id = d.message_id
			),
			deliveries_filtered_by_condition(id, delivery_ts, status, dsn, queue_id, message_id, direction, returned, mailfrom, mailto, relay_id) as (
				select id, delivery_ts, status, dsn, queue_id, message_id, direction, returned, mailfrom, mailto, relay_id from sent_deliveries_filtered_by_condition
				union
				select id, delivery_ts, status, dsn, queue_id, message_id, direction, returned, mailfrom, mailto, relay_id from returned_deliveries
			),
			queues_filtered_by_condition(delivery_id, queue_id, expired_ts, mailfrom, mailto) as (
				select distinct deliveries_filtered_by_condition.id, delivery_queue.queue_id, expired_ts, mailfrom, mailto
				from deliveries_filtered_by_condition
				left join expired_queues eq on eq.queue_id = deliveries_filtered_by_condition.queue_id
				join delivery_queue on delivery_queue.delivery_id = deliveries_filtered_by_condition.id
			),
			grouped_and_computed(log_refs, rn, total, delivery_ts, status, dsn, queue_id, message_id, queue, expired_ts, number_of_attempts, min_ts, max_ts, direction, returned, mailfrom, mailto, relay) as (
				select
					json_group_array(distinct iif(ref.time is null, json_object('invalid', true), json_object('time', ref.time, 'checksum', ref.checksum))),
					row_number() over (order by delivery_ts),
					count() over () as total,
					delivery_ts, status, dsn, d.queue_id, d.message_id, queues.name as queue, expired_ts,
					count(distinct delivery_ts) as number_of_attempts, min(delivery_ts) as min_ts, max(delivery_ts) as max_ts,
					d.direction as direction,
					d.returned as returned,
					d.mailfrom, json_group_array(distinct d.mailto),
					json_group_array(distinct lm_host_domain_from_domain(coalesce(next_relays.hostname, 'local')))
				from deliveries_filtered_by_condition d
				join queues on d.queue_id = queues.id
				join queues_filtered_by_condition q on q.queue_id = d.queue_id 
				left join next_relays on d.relay_id = next_relays.id
				left join log_lines_ref ref on d.id = ref.delivery_id and (ref.ref_type = ? or ref.ref_type is null)
				group by d.queue_id, status, dsn
			)
			select total, status, dsn, queue, message_id, expired_ts, number_of_attempts, min_ts, max_ts, direction, returned, mailfrom, mailto, relay, log_refs
			from grouped_and_computed gac
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

	if err := deliveriesConnPool.ForEach(setup); err != nil {
		return nil, errorutil.Wrap(err)
	}

	return &sqlDetective{
		deliveriesConnPool: deliveriesConnPool,
		rawLogsAccessor:    rawLogsAccessor,
	}, nil
}

var ErrNoAvailableLogs = errors.New(`No available logs`)

func (d *sqlDetective) CheckMessageDelivery(ctx context.Context, mailFrom string, mailTo string, interval timeutil.TimeInterval, status int, someID string, page int) (*MessagesPage, error) {
	conn, release, err := d.deliveriesConnPool.AcquireContext(ctx)
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	defer release()

	//nolint:sqlclosecheck
	return checkMessageDelivery(ctx, d.rawLogsAccessor, conn.GetStmt(checkMessageDeliveryKey), mailFrom, mailTo, interval, status, someID, page)
}

func (d *sqlDetective) OldestAvailableTime(ctx context.Context) (time.Time, error) {
	conn, release, err := d.deliveriesConnPool.AcquireContext(ctx)
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
	Queue     QueueName         `json:"queue"`
	MessageID string            `json:"message_id"`
	Entries   []MessageDelivery `json:"entries"`
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

	status, err := parser.ParseStatus(v)
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
	Relays           []string   `json:"relays"`
	Expired          *time.Time `json:"expired"`
	MailFrom         string     `json:"from"`
	MailTo           []string   `json:"to"`
	RawLogMsgs       []string   `json:"log_msgs"`
}

func parseLogRefs(ctx context.Context, rawLogsAccessor rawlogsdb.Accessor, content string) ([]string, error) {
	var logRefs []struct {
		Time     int64       `json:"time"`
		Checksum postfix.Sum `json:"checksum"`
		Invalid  int         `json:"invalid"`
	}

	if err := json.Unmarshal([]byte(content), &logRefs); err != nil {
		return nil, errorutil.Wrap(err)
	}

	logLines := make([]string, 0, len(logRefs))

	for _, ref := range logRefs {
		// NOTE: this is a very ugly hack!
		if ref.Invalid == 1 {
			continue
		}

		logLine, err := rawLogsAccessor.FetchLogLine(ctx, time.Unix(ref.Time, 0), ref.Checksum)
		if err != nil && !errors.Is(err, rawlogsdb.ErrLogLineNotFound) {
			return nil, errorutil.Wrap(err)
		}

		logLines = append(logLines, logLine)
	}

	return logLines, nil
}

// NOTE: we are checking rows.Err(), but the linter won't see that
//nolint:gocognit
func checkMessageDelivery(ctx context.Context, rawLogsAccessor rawlogsdb.Accessor, stmt *sql.Stmt, mailFrom string, mailTo string, interval timeutil.TimeInterval, status int, someID string, page int) (messagesPage *MessagesPage, err error) {
	splitEmail := func(email string) (local, domain string, err error) {
		if len(email) == 0 {
			return "", "", nil
		}

		local, domain, _, err = emailutil.SplitPartial(email)
		if err != nil {
			return "", "", errorutil.Wrap(err)
		}

		return local, domain, nil
	}

	senderLocal, senderDomain, err := splitEmail(mailFrom)
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	recipientLocal, recipientDomain, err := splitEmail(mailTo)
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	queryStart := time.Now()

	defer func() {
		log.Debug().Msgf("Time to execute checkMessageDelivery: %v", time.Since(queryStart))
	}()

	//nolint:sqlclosecheck
	rows, err := stmt.QueryContext(ctx,
		senderLocal, senderLocal,
		senderDomain, senderDomain,
		recipientLocal, recipientLocal,
		recipientDomain, fmt.Sprintf("%%%s", recipientDomain), recipientDomain,
		interval.From.Unix(), interval.To.Unix(),
		status, status, status, status,
		someID, someID, someID,
		tracking.ResultDeliveryLineChecksum,
		resultsPerPage, (page-1)*resultsPerPage,
	)

	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	defer errorutil.UpdateErrorFromCloser(rows, &err)

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
			messageID        string
			expiredTs        *int64
			expiredTime      *time.Time
			numberOfAttempts int
			tsMin            int64
			tsMax            int64
			direction        int64
			returned         bool
			mailFrom         string
			mailTo           string
			relay            string
			logRefsContent   string
		)

		if err := rows.Scan(&total, &status, &dsn, &queueName, &messageID, &expiredTs, &numberOfAttempts, &tsMin, &tsMax, &direction, &returned, &mailFrom, &mailTo, &relay, &logRefsContent); err != nil {
			return nil, errorutil.Wrap(err)
		}

		if tracking.MessageDirection(direction) == tracking.MessageDirectionIncoming {
			status = parser.ReceivedStatus
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

			messages = append(messages, Message{Queue: queueName, MessageID: messageID, Entries: []MessageDelivery{}})

			return len(messages) - 1
		}()

		if expiredTs != nil {
			eT := time.Unix(*expiredTs, 0).In(time.UTC)
			expiredTime = &eT
		}

		var mailTos []string
		if err := json.Unmarshal([]byte(mailTo), &mailTos); err != nil {
			return nil, errorutil.Wrap(err)
		}

		var relays []string
		if err := json.Unmarshal([]byte(relay), &relays); err != nil {
			return nil, errorutil.Wrap(err)
		}

		var logRefs []struct {
			Time     int64       `json:"time"`
			Checksum postfix.Sum `json:"checksum"`
			Invalid  int         `json:"invalid"`
		}

		if err := json.Unmarshal([]byte(logRefsContent), &logRefs); err != nil {
			return nil, errorutil.Wrap(err)
		}

		logLines, err := parseLogRefs(ctx, rawLogsAccessor, logRefsContent)
		if err != nil {
			return nil, errorutil.Wrap(err)
		}

		delivery := MessageDelivery{
			NumberOfAttempts: numberOfAttempts,
			TimeMin:          time.Unix(tsMin, 0).In(time.UTC),
			TimeMax:          time.Unix(tsMax, 0).In(time.UTC),
			Status:           Status(status),
			Dsn:              dsn,
			Expired:          expiredTime,
			MailFrom:         mailFrom,
			MailTo:           mailTos,
			Relays:           relays,
			RawLogMsgs:       logLines,
		}

		messages[index].Entries = append(messages[index].Entries, delivery)
	}

	if err := rows.Err(); err != nil {
		return nil, errorutil.Wrap(err)
	}

	return &MessagesPage{
		PageNumber:   page,
		FirstPage:    1,
		LastPage:     total/resultsPerPage + 1,
		TotalResults: total - grouped,
		Messages:     messages,
	}, nil
}
