// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0

package dashboard

import (
	"context"
	"database/sql"
	"errors"
	"gitlab.com/lightmeter/controlcenter/data"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	parser "gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser"
	"gitlab.com/lightmeter/controlcenter/tracking"
	"gitlab.com/lightmeter/controlcenter/util/closeutil"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"strings"
)

type queries struct {
	countByStatus      *sql.Stmt
	deliveryStatus     *sql.Stmt
	topBusiestDomains  *sql.Stmt
	topDomainsByStatus *sql.Stmt
}

type Pair struct {
	Key   interface{} `json:"key"`
	Value interface{} `json:"value"`
}

type Pairs []Pair

type Dashboard interface {
	Close() error

	CountByStatus(context.Context, parser.SmtpStatus, data.TimeInterval) (int, error)
	TopBusiestDomains(context.Context, data.TimeInterval) (Pairs, error)
	TopBouncedDomains(context.Context, data.TimeInterval) (Pairs, error)
	TopDeferredDomains(context.Context, data.TimeInterval) (Pairs, error)
	DeliveryStatus(context.Context, data.TimeInterval) (Pairs, error)
}

type sqlDashboard struct {
	queries queries
	closers closeutil.Closers
}

func New(db dbconn.RoConn) (Dashboard, error) {
	countByStatus, err := db.Prepare(`
	select
		count(*)
	from
		deliveries
	where
		status = ? and delivery_ts between ? and ? and direction = ?`)

	if err != nil {
		return nil, errorutil.Wrap(err)
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
		delivery_ts between ? and ? and direction = ?
	group by
		status
	order by
		status
	`)

	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	defer func() {
		if err != nil {
			errorutil.MustSucceed(deliveryStatus.Close(), "Closing deliveryStatus")
		}
	}()

	domainMappingByRecipientDomainPartStmtPart := `
with resolve_domain_mapping_view(domain, status, direction, delivery_ts)
as
(
with
	aux_domain_mapping(orig_domain, domain_mapped_to, status, direction, delivery_ts)
as (
select
	remote_domains.domain, temp_domain_mapping.mapped, deliveries.status, deliveries.direction, deliveries.delivery_ts
from
	deliveries join remote_domains on deliveries.recipient_domain_part_id = remote_domains.rowid
	left join temp_domain_mapping on remote_domains.domain = temp_domain_mapping.orig
) select
	ifnull(domain_mapped_to, orig_domain) as domain, status, direction, delivery_ts
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
                status = ? and delivery_ts between ? and ? and direction = ?
        group by
                domain collate nocase
        order by
                c desc, domain collate nocase asc
        limit 20
	`)

	if err != nil {
		return nil, errorutil.Wrap(err)
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
                delivery_ts between ? and ? and direction = ?
        group by
                domain collate nocase
        order by
                c desc, domain collate nocase asc
        limit 20
	`)

	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	defer func() {
		if err != nil {
			errorutil.MustSucceed(topBusiestDomains.Close(), "Closing topBusiestDomains")
		}
	}()

	return &sqlDashboard{
		queries: queries{
			countByStatus:      countByStatus,
			deliveryStatus:     deliveryStatus,
			topBusiestDomains:  topBusiestDomains,
			topDomainsByStatus: topDomainsByStatus,
		},
		closers: closeutil.New(
			countByStatus,
			deliveryStatus,
			topBusiestDomains,
			topDomainsByStatus,
		),
	}, nil
}

var ErrClosingDashboardQueries = errors.New("Error closing any of the dashboard queries")

func (d *sqlDashboard) Close() error {
	if err := d.closers.Close(); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func (d sqlDashboard) CountByStatus(ctx context.Context, status parser.SmtpStatus, interval data.TimeInterval) (int, error) {
	return countByStatus(ctx, d.queries.countByStatus, status, interval)
}

func (d sqlDashboard) TopBusiestDomains(ctx context.Context, interval data.TimeInterval) (Pairs, error) {
	return listDomainAndCount(ctx, d.queries.topBusiestDomains, interval.From.Unix(), interval.To.Unix(), tracking.MessageDirectionOutbound)
}

func (d sqlDashboard) TopBouncedDomains(ctx context.Context, interval data.TimeInterval) (Pairs, error) {
	return listDomainAndCount(ctx, d.queries.topDomainsByStatus, parser.BouncedStatus,
		interval.From.Unix(), interval.To.Unix(), tracking.MessageDirectionOutbound)
}

func (d sqlDashboard) TopDeferredDomains(ctx context.Context, interval data.TimeInterval) (Pairs, error) {
	return listDomainAndCount(ctx, d.queries.topDomainsByStatus, parser.DeferredStatus,
		interval.From.Unix(), interval.To.Unix(), tracking.MessageDirectionOutbound)
}

func (d sqlDashboard) DeliveryStatus(ctx context.Context, interval data.TimeInterval) (Pairs, error) {
	return deliveryStatus(ctx, d.queries.deliveryStatus, interval)
}

func countByStatus(ctx context.Context, stmt *sql.Stmt, status parser.SmtpStatus, interval data.TimeInterval) (int, error) {
	countValue := 0

	if err := stmt.QueryRowContext(ctx, status, interval.From.Unix(), interval.To.Unix(), tracking.MessageDirectionOutbound).
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
func deliveryStatus(ctx context.Context, stmt *sql.Stmt, interval data.TimeInterval) (Pairs, error) {
	r := Pairs{}

	query, err := stmt.QueryContext(ctx, interval.From.Unix(), interval.To.Unix(), tracking.MessageDirectionOutbound)

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
