// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package dashboard

import (
	"context"
	"database/sql"
	"encoding/json"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	parser "gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser"
	"gitlab.com/lightmeter/controlcenter/tracking"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"io"
	"math"
	"strings"
	"time"
)

type Pair struct {
	Key   interface{} `json:"key"`
	Value interface{} `json:"value"`
}

type Pairs []Pair

type MailTrafficPerSenderOverTimeResult struct {
	Times  []int64            `json:"times"`
	Values map[string][]int64 `json:"values"`
}

type Dashboard interface {
	CountByStatus(context.Context, parser.SmtpStatus, timeutil.TimeInterval) (int, error)
	TopBusiestDomains(context.Context, timeutil.TimeInterval) (Pairs, error)
	TopBouncedDomains(context.Context, timeutil.TimeInterval) (Pairs, error)
	TopDeferredDomains(context.Context, timeutil.TimeInterval) (Pairs, error)
	DeliveryStatus(context.Context, timeutil.TimeInterval) (Pairs, error)
	SentMailsByMailbox(context.Context, timeutil.TimeInterval, int) (MailTrafficPerSenderOverTimeResult, error)
	BouncedMailsByMailbox(context.Context, timeutil.TimeInterval, int) (MailTrafficPerSenderOverTimeResult, error)
	DeferredMailsByMailbox(context.Context, timeutil.TimeInterval, int) (MailTrafficPerSenderOverTimeResult, error)
	ExpiredMailsByMailbox(context.Context, timeutil.TimeInterval, int) (MailTrafficPerSenderOverTimeResult, error)
	ReceivedMailsByMailbox(context.Context, timeutil.TimeInterval, int) (MailTrafficPerSenderOverTimeResult, error)
}

type sqlDashboard struct {
	pool *dbconn.RoPool
}

// direction: 0 is outbound, 1 is inbound (as defined by the tracking package)
const directionQueryFragment = ` and (direction = 0 || (direction = 1 and sender_domain_part_id = recipient_domain_part_id))`

func New(pool *dbconn.RoPool) (Dashboard, error) {
	setup := func(db *dbconn.RoPooledConn) error {
		if err := db.PrepareStmt(`
	select
		count(*)
	from
		deliveries
	where
		status = ? and delivery_ts between ? and ?`+directionQueryFragment, "countByStatus"); err != nil {
			return errorutil.Wrap(err)
		}

		if err := db.PrepareStmt(`
	select
		status, count(status) as c
	from
		deliveries
	where
		delivery_ts between ? and ?`+directionQueryFragment+`
	group by
		status
	order by
		status
	`, "deliveryStatus"); err != nil {
			return errorutil.Wrap(err)
		}

		domainMappingByRecipientDomainPartStmtPart := `
with
aux_domain_mapping(orig_domain, domain_mapped_to, status, direction, sender_domain_part_id, recipient_domain_part_id, delivery_ts)
as (
select
	remote_domains.domain, temp_domain_mapping.mapped, deliveries.status,
	deliveries.direction, deliveries.sender_domain_part_id, deliveries.recipient_domain_part_id, deliveries.delivery_ts
from
	deliveries join remote_domains on deliveries.recipient_domain_part_id = remote_domains.rowid
	left join temp_domain_mapping on remote_domains.domain = temp_domain_mapping.orig
),
resolve_domain_mapping_view(domain, status, direction, sender_domain_part_id, recipient_domain_part_id, delivery_ts)
as (
 select
	coalesce(domain_mapped_to, orig_domain) as domain, status, direction, sender_domain_part_id, recipient_domain_part_id, delivery_ts
from
	aux_domain_mapping
)
`

		if err := db.PrepareStmt(domainMappingByRecipientDomainPartStmtPart+`
				select
                domain, count(domain) as c
        from
                resolve_domain_mapping_view
        where
                status = ? and delivery_ts between ? and ?`+directionQueryFragment+`
        group by
                domain collate nocase
        order by
                c desc, domain collate nocase asc
        limit 20
	`, "topDomainsByStatus"); err != nil {
			return errorutil.Wrap(err)
		}

		if err := db.PrepareStmt(domainMappingByRecipientDomainPartStmtPart+`
				select
                domain, count(domain) as c
        from
                resolve_domain_mapping_view
        where
                delivery_ts between ? and ? `+directionQueryFragment+`
        group by
                domain collate nocase
        order by
                c desc, domain collate nocase asc
        limit 20
	`, "topBusiestDomains"); err != nil {
			return errorutil.Wrap(err)
		}

		if err := db.PrepareStmt(`
			with deliveries_in_range as (
				select * from deliveries where delivery_ts between @start and @end
			),
			users as (
				select distinct
						case d.direction when @Outbound then d.sender_local_part     else d.recipient_local_part     end as local_part,
						case d.direction when @Outbound then d.sender_domain_part_id else d.recipient_domain_part_id end as domain_part_id,
						rd.domain
				from
					deliveries_in_range d join remote_domains rd on
						@direction = @Outbound and d.sender_domain_part_id = rd.id    and sender_local_part != ''
						or
						@direction = @Inbound  and d.recipient_domain_part_id = rd.id and recipient_local_part != ''
				where
					d.direction = @direction and domain != ''
			),
			bins as (
				select
					-- this round is equivalent to floor(), but using built-in functions (floor requires an external build flag)
					cast(round(delivery_ts/(@granularity), 0.5)*(@granularity) as integer) as t,
					id,
					u.local_part,
					u.domain
				from
					deliveries_in_range d join users u on
						@direction = @Outbound and d.sender_local_part    = u.local_part and d.sender_domain_part_id    = u.domain_part_id
						or
						@direction = @Inbound  and d.recipient_local_part = u.local_part and d.recipient_domain_part_id = u.domain_part_id
				where
					status = @status
					or @status = @Expired and exists(
						select *
						from expired_queues eq 
						join delivery_queue dq on eq.queue_id = dq.queue_id
						where delivery_id = d.id
					)
				order by
						t
			),
			number_sent_mails_per_user_per_interval as (
				select
					t, count(id) as c, local_part || '@' || domain as mailbox
				from
					bins
				group by
					t, local_part, domain
			)
			select
				mailbox, min(t) as min_r, max(t) as max_r, json_group_array(json_array(t, c))
			from
				number_sent_mails_per_user_per_interval
			group by mailbox
			order by t`, "outboundSentVolumeByMailbox"); err != nil {
			return errorutil.Wrap(err)
		}

		return nil
	}

	if err := pool.ForEach(setup); err != nil {
		return nil, errorutil.Wrap(err)
	}

	return &sqlDashboard{
		pool: pool,
	}, nil
}

func (d sqlDashboard) CountByStatus(ctx context.Context, status parser.SmtpStatus, interval timeutil.TimeInterval) (int, error) {
	conn, release, err := d.pool.AcquireContext(ctx)
	if err != nil {
		return 0, errorutil.Wrap(err)
	}

	defer release()

	//nolint:sqlclosecheck
	return countByStatus(ctx, conn.GetStmt("countByStatus"), status, interval)
}

func (d sqlDashboard) TopBusiestDomains(ctx context.Context, interval timeutil.TimeInterval) (Pairs, error) {
	conn, release, err := d.pool.AcquireContext(ctx)
	if err != nil {
		return Pairs{}, errorutil.Wrap(err)
	}

	defer release()

	//nolint:sqlclosecheck
	return listDomainAndCount(ctx, conn.GetStmt("topBusiestDomains"), interval.From.Unix(), interval.To.Unix())
}

func (d sqlDashboard) TopBouncedDomains(ctx context.Context, interval timeutil.TimeInterval) (Pairs, error) {
	conn, release, err := d.pool.AcquireContext(ctx)
	if err != nil {
		return Pairs{}, errorutil.Wrap(err)
	}

	defer release()

	//nolint:sqlclosecheck
	return listDomainAndCount(ctx, conn.GetStmt("topDomainsByStatus"), parser.BouncedStatus,
		interval.From.Unix(), interval.To.Unix())
}

func (d sqlDashboard) TopDeferredDomains(ctx context.Context, interval timeutil.TimeInterval) (Pairs, error) {
	conn, release, err := d.pool.AcquireContext(ctx)
	if err != nil {
		return Pairs{}, errorutil.Wrap(err)
	}

	defer release()

	//nolint:sqlclosecheck
	return listDomainAndCount(ctx, conn.GetStmt("topDomainsByStatus"), parser.DeferredStatus,
		interval.From.Unix(), interval.To.Unix())
}

func (d sqlDashboard) DeliveryStatus(ctx context.Context, interval timeutil.TimeInterval) (Pairs, error) {
	conn, release, err := d.pool.AcquireContext(ctx)
	if err != nil {
		return Pairs{}, errorutil.Wrap(err)
	}

	defer release()

	//nolint:sqlclosecheck
	return deliveryStatus(ctx, conn.GetStmt("deliveryStatus"), interval)
}

func (d sqlDashboard) getVolumesBySender(ctx context.Context, interval timeutil.TimeInterval, granularityInHour int, status parser.SmtpStatus, direction tracking.MessageDirection) (MailTrafficPerSenderOverTimeResult, error) {
	conn, release, err := d.pool.AcquireContext(ctx)
	if err != nil {
		return MailTrafficPerSenderOverTimeResult{}, errorutil.Wrap(err)
	}

	defer release()

	granularity := granularityInHour * int(time.Hour/time.Second)

	//nolint:sqlclosecheck
	return queryMailTrafficPerMailbox(ctx, conn.GetStmt("outboundSentVolumeByMailbox"),
		granularity,

		sql.Named("Outbound", tracking.MessageDirectionOutbound),
		sql.Named("Inbound", tracking.MessageDirectionIncoming),
		sql.Named("Expired", parser.ExpiredStatus),

		sql.Named("start", interval.From.Unix()),
		sql.Named("end", interval.To.Unix()),
		sql.Named("status", status),
		sql.Named("direction", direction))
}

func (d sqlDashboard) SentMailsByMailbox(ctx context.Context, interval timeutil.TimeInterval, granularityInHour int) (MailTrafficPerSenderOverTimeResult, error) {
	return d.getVolumesBySender(ctx, interval, granularityInHour, parser.SentStatus, tracking.MessageDirectionOutbound)
}

func (d sqlDashboard) BouncedMailsByMailbox(ctx context.Context, interval timeutil.TimeInterval, granularityInHour int) (MailTrafficPerSenderOverTimeResult, error) {
	return d.getVolumesBySender(ctx, interval, granularityInHour, parser.BouncedStatus, tracking.MessageDirectionOutbound)
}

func (d sqlDashboard) DeferredMailsByMailbox(ctx context.Context, interval timeutil.TimeInterval, granularityInHour int) (MailTrafficPerSenderOverTimeResult, error) {
	return d.getVolumesBySender(ctx, interval, granularityInHour, parser.DeferredStatus, tracking.MessageDirectionOutbound)
}

func (d sqlDashboard) ExpiredMailsByMailbox(ctx context.Context, interval timeutil.TimeInterval, granularityInHour int) (MailTrafficPerSenderOverTimeResult, error) {
	return d.getVolumesBySender(ctx, interval, granularityInHour, parser.ExpiredStatus, tracking.MessageDirectionOutbound)
}

func (d sqlDashboard) ReceivedMailsByMailbox(ctx context.Context, interval timeutil.TimeInterval, granularityInHour int) (MailTrafficPerSenderOverTimeResult, error) {
	return d.getVolumesBySender(ctx, interval, granularityInHour, parser.SentStatus /* <= not a bug*/, tracking.MessageDirectionIncoming)
}

type queryable interface {
	QueryContext(ctx context.Context, args ...interface{}) (queryableRows, error)
}

type queryableRows interface {
	io.Closer
	ForEach(func(queryableScanner) error) error
}

type queryableScanner interface {
	Scan(...interface{}) error
}

type realQueryableScanner struct {
	rows *sql.Rows
}

func (s *realQueryableScanner) Scan(args ...interface{}) error {
	return s.rows.Scan(args...)
}

type realQueryableRows struct {
	*sql.Rows
}

func (r *realQueryableRows) ForEach(f func(s queryableScanner) error) error {
	scanner := realQueryableScanner{rows: r.Rows}

	for r.Rows.Next() {
		if err := f(&scanner); err != nil {
			return errorutil.Wrap(err)
		}
	}

	if err := r.Rows.Err(); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

type realStmtWrap struct {
	stmt *sql.Stmt
}

func (s *realStmtWrap) QueryContext(ctx context.Context, args ...interface{}) (queryableRows, error) {
	rows, err := s.stmt.QueryContext(ctx, args...)
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	return &realQueryableRows{
		Rows: rows,
	}, nil
}

func wrapStmt(stmt *sql.Stmt) queryable {
	return &realStmtWrap{stmt: stmt}
}

func queryMailTrafficPerMailbox(ctx context.Context, stmt *sql.Stmt, granularity int, args ...interface{}) (result MailTrafficPerSenderOverTimeResult, err error) {
	return queryMailTrafficPerMailboxWithQueryable(ctx, wrapStmt(stmt), granularity, args...)
}

func queryMailTrafficPerMailboxWithQueryable(ctx context.Context, stmt queryable, granularity int, args ...interface{}) (result MailTrafficPerSenderOverTimeResult, err error) {
	//nolint:sqlclosecheck
	rows, err := stmt.QueryContext(ctx, append([]interface{}{sql.Named("granularity", granularity)}, args...)...)

	if err != nil {
		return MailTrafficPerSenderOverTimeResult{}, errorutil.Wrap(err)
	}

	defer errorutil.UpdateErrorFromCloser(rows, &err)

	type counter [2]int64

	basicResult := map[string][]counter{}

	var (
		// NOTE: we start with extreme values, obviously
		overallMinTime int64 = math.MaxInt64
		overallMaxTime int64 = math.MinInt64
	)

	if err := rows.ForEach(func(scanner queryableScanner) error {
		var (
			mailbox          string
			counters         []counter
			rawCounters      string
			minTime, maxTime int64
		)

		if err := scanner.Scan(&mailbox, &minTime, &maxTime, &rawCounters); err != nil {
			return errorutil.Wrap(err)
		}

		if err := json.Unmarshal([]byte(rawCounters), &counters); err != nil {
			return errorutil.Wrap(err)
		}

		basicResult[mailbox] = counters

		overallMinTime = min(minTime, overallMinTime)
		overallMaxTime = max(maxTime, overallMaxTime)

		return nil
	}); err != nil {
		return MailTrafficPerSenderOverTimeResult{}, errorutil.Wrap(err)
	}

	compute := func(min, max int64) int64 {
		return int64(float64(max-min) / float64(granularity))
	}

	resultLen := compute(overallMinTime, overallMaxTime) + 1

	times := make([]int64, resultLen)

	for t := overallMinTime; t < overallMaxTime; t += int64(granularity) {
		i := compute(overallMinTime, t)
		times[i] = t
	}

	values := map[string][]int64{}

	for k, v := range basicResult {
		counters := make([]int64, resultLen)

		for _, c := range v {
			i := compute(overallMinTime, c[0])
			counters[i] = c[1]
		}

		values[k] = counters
	}

	return MailTrafficPerSenderOverTimeResult{
		Times:  times,
		Values: values,
	}, nil
}

// FIXME: yes, this is ugly
func min(a, b int64) int64 {
	if a < b {
		return a
	}

	return b
}

// FIXME: yes, this is ugly
func max(a, b int64) int64 {
	if a > b {
		return a
	}

	return b
}

func countByStatus(ctx context.Context, stmt *sql.Stmt, status parser.SmtpStatus, interval timeutil.TimeInterval) (int, error) {
	countValue := 0

	if err := stmt.QueryRowContext(ctx, status, interval.From.Unix(), interval.To.Unix()).
		Scan(&countValue); err != nil {
		return 0, errorutil.Wrap(err)
	}

	return countValue, nil
}

func listDomainAndCount(ctx context.Context, stmt *sql.Stmt, args ...interface{}) (r Pairs, err error) {
	//nolint:sqlclosecheck
	query, err := stmt.QueryContext(ctx, args...)
	if err != nil {
		return Pairs{}, errorutil.Wrap(err)
	}

	defer errorutil.UpdateErrorFromCloser(query, &err)

	for query.Next() {
		var (
			domain     string
			countValue int
		)

		err = query.Scan(&domain, &countValue)

		if err != nil {
			return Pairs{}, errorutil.Wrap(err)
		}

		// If the relay info is not available, use a placeholder
		if len(domain) == 0 {
			domain = "<none>"
		}

		r = append(r, Pair{strings.ToLower(domain), countValue})
	}

	if err := query.Err(); err != nil {
		return Pairs{}, errorutil.Wrap(err)
	}

	return r, nil
}

func deliveryStatus(ctx context.Context, stmt *sql.Stmt, interval timeutil.TimeInterval) (r Pairs, err error) {
	//nolint:sqlclosecheck
	query, err := stmt.QueryContext(ctx, interval.From.Unix(), interval.To.Unix())
	if err != nil {
		return Pairs{}, errorutil.Wrap(err)
	}

	defer errorutil.UpdateErrorFromCloser(query, &err)

	for query.Next() {
		var (
			status parser.SmtpStatus
			value  int
		)

		err = query.Scan(&status, &value)

		if err != nil {
			return Pairs{}, errorutil.Wrap(err)
		}

		r = append(r, Pair{status.String(), value})
	}

	if err := query.Err(); err != nil {
		return Pairs{}, errorutil.Wrap(err)
	}

	return r, nil
}
