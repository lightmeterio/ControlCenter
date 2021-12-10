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
