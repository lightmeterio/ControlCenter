// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package dashboard

import (
	"context"
	"database/sql"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	parser "gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"strings"
)

type Pair struct {
	Key   interface{} `json:"key"`
	Value interface{} `json:"value"`
}

type Pairs []Pair

type Dashboard interface {
	CountByStatus(context.Context, parser.SmtpStatus, timeutil.TimeInterval) (int, error)
	TopBusiestDomains(context.Context, timeutil.TimeInterval) (Pairs, error)
	TopBouncedDomains(context.Context, timeutil.TimeInterval) (Pairs, error)
	TopDeferredDomains(context.Context, timeutil.TimeInterval) (Pairs, error)
	DeliveryStatus(context.Context, timeutil.TimeInterval) (Pairs, error)
}

type sqlDashboard struct {
	pool *dbconn.RoPool
}

// direction: 0 is outbound, 1 is inbound (as defined by the tracking package)
const directionQueryFragment = ` and (direction = 0 || (direction = 1 and sender_domain_part_id = recipient_domain_part_id))`

func New(pool *dbconn.RoPool) (Dashboard, error) {
	setup := func(db *dbconn.RoPooledConn) error {
		countByStatus, err := db.Prepare(`
	select
		count(*)
	from
		deliveries
	where
		status = ? and delivery_ts between ? and ?` + directionQueryFragment)

		if err != nil {
			return errorutil.Wrap(err)
		}

		defer func() {
			if err != nil {
				errorutil.MustSucceed(countByStatus.Close(), "Closing countByStatus")
			}
		}()

		deliveryStatus, err := db.Prepare(`
	select
		status, count(status) as c
	from
		deliveries
	where
		delivery_ts between ? and ?` + directionQueryFragment + `
	group by
		status
	order by
		status
	`)

		if err != nil {
			return errorutil.Wrap(err)
		}

		defer func() {
			if err != nil {
				errorutil.MustSucceed(deliveryStatus.Close(), "Closing deliveryStatus")
			}
		}()

		domainMappingByRecipientDomainPartStmtPart := `
with resolve_domain_mapping_view(domain, status, direction, sender_domain_part_id, recipient_domain_part_id, delivery_ts)
as
(
with
	aux_domain_mapping(orig_domain, domain_mapped_to, status, direction, sender_domain_part_id, recipient_domain_part_id, delivery_ts)
as (
select
	remote_domains.domain, temp_domain_mapping.mapped, deliveries.status,
	deliveries.direction, deliveries.sender_domain_part_id, deliveries.recipient_domain_part_id, deliveries.delivery_ts
from
	deliveries join remote_domains on deliveries.recipient_domain_part_id = remote_domains.rowid
	left join temp_domain_mapping on remote_domains.domain = temp_domain_mapping.orig
) select
	ifnull(domain_mapped_to, orig_domain) as domain, status, direction, sender_domain_part_id, recipient_domain_part_id, delivery_ts
from
	aux_domain_mapping
)
`

		topDomainsByStatus, err := db.Prepare(domainMappingByRecipientDomainPartStmtPart + `
				select
                domain, count(domain) as c
        from
                resolve_domain_mapping_view
        where
                status = ? and delivery_ts between ? and ?` + directionQueryFragment + `
        group by
                domain collate nocase
        order by
                c desc, domain collate nocase asc
        limit 20
	`)

		if err != nil {
			return errorutil.Wrap(err)
		}

		defer func() {
			if err != nil {
				errorutil.MustSucceed(topDomainsByStatus.Close(), "Closing topDomainsByStatus")
			}
		}()

		topBusiestDomains, err := db.Prepare(domainMappingByRecipientDomainPartStmtPart + `
				select
                domain, count(domain) as c
        from
                resolve_domain_mapping_view
        where
                delivery_ts between ? and ? ` + directionQueryFragment + `
        group by
                domain collate nocase
        order by
                c desc, domain collate nocase asc
        limit 20
	`)

		if err != nil {
			return errorutil.Wrap(err)
		}

		defer func() {
			if err != nil {
				errorutil.MustSucceed(topBusiestDomains.Close(), "Closing topBusiestDomains")
			}
		}()

		db.Closers.Add(countByStatus, deliveryStatus, topBusiestDomains, topDomainsByStatus)

		db.Stmts["countByStatus"] = countByStatus
		db.Stmts["deliveryStatus"] = deliveryStatus
		db.Stmts["topBusiestDomains"] = topBusiestDomains
		db.Stmts["topDomainsByStatus"] = topDomainsByStatus

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
	conn, release := d.pool.Acquire()

	defer release()

	return countByStatus(ctx, conn.Stmts["countByStatus"], status, interval)
}

func (d sqlDashboard) TopBusiestDomains(ctx context.Context, interval timeutil.TimeInterval) (Pairs, error) {
	conn, release := d.pool.Acquire()

	defer release()

	return listDomainAndCount(ctx, conn.Stmts["topBusiestDomains"], interval.From.Unix(), interval.To.Unix())
}

func (d sqlDashboard) TopBouncedDomains(ctx context.Context, interval timeutil.TimeInterval) (Pairs, error) {
	conn, release := d.pool.Acquire()

	defer release()

	return listDomainAndCount(ctx, conn.Stmts["topDomainsByStatus"], parser.BouncedStatus,
		interval.From.Unix(), interval.To.Unix())
}

func (d sqlDashboard) TopDeferredDomains(ctx context.Context, interval timeutil.TimeInterval) (Pairs, error) {
	conn, release := d.pool.Acquire()

	defer release()

	return listDomainAndCount(ctx, conn.Stmts["topDomainsByStatus"], parser.DeferredStatus,
		interval.From.Unix(), interval.To.Unix())
}

func (d sqlDashboard) DeliveryStatus(ctx context.Context, interval timeutil.TimeInterval) (Pairs, error) {
	conn, release := d.pool.Acquire()

	defer release()

	return deliveryStatus(ctx, conn.Stmts["deliveryStatus"], interval)
}

func countByStatus(ctx context.Context, stmt *sql.Stmt, status parser.SmtpStatus, interval timeutil.TimeInterval) (int, error) {
	countValue := 0

	if err := stmt.QueryRowContext(ctx, status, interval.From.Unix(), interval.To.Unix()).
		Scan(&countValue); err != nil {
		return 0, errorutil.Wrap(err)
	}

	return countValue, nil
}

// rowserrcheck is buggy and unable to see that the query errors are being checked
// when query.Close() is inside a closure
//nolint:rowserrcheck
func listDomainAndCount(ctx context.Context, stmt *sql.Stmt, args ...interface{}) (Pairs, error) {
	r := Pairs{}

	query, err := stmt.QueryContext(ctx, args...)

	if err != nil {
		return Pairs{}, errorutil.Wrap(err)
	}

	defer func() { errorutil.MustSucceed(query.Close()) }()

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

// rowserrcheck is buggy and unable to see that the query errors are being checked
// when query.Close() is inside a closure
//nolint:rowserrcheck
func deliveryStatus(ctx context.Context, stmt *sql.Stmt, interval timeutil.TimeInterval) (Pairs, error) {
	r := Pairs{}

	query, err := stmt.QueryContext(ctx, interval.From.Unix(), interval.To.Unix())

	if err != nil {
		return Pairs{}, errorutil.Wrap(err)
	}

	defer func() { errorutil.MustSucceed(query.Close()) }()

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
