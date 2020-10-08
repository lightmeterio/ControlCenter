package dashboard

import (
	"context"
	"database/sql"
	"errors"
	"gitlab.com/lightmeter/controlcenter/data"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/util/closeutil"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	parser "gitlab.com/lightmeter/postfix-log-parser"
	"strings"
)

type queries struct {
	countByStatus      *sql.Stmt
	deliveryStatus     *sql.Stmt
	topBusiestDomains  *sql.Stmt
	topDomainsByStatus *sql.Stmt
}

type Pair struct {
	Key   interface{}
	Value interface{}
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

const removeSentToLocalhostSqlFragment = `((process_ip is not null and relay_ip != process_ip) or (process_ip is null and relay_name != "127.0.0.1"))`

func New(db dbconn.RoConn) (Dashboard, error) {
	countByStatus, err := db.Prepare(`
	select
		count(*)
	from
		postfix_smtp_message_status
	where
		status = ? and read_ts_sec between ? and ? and ` + removeSentToLocalhostSqlFragment)

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
		postfix_smtp_message_status
	where
		read_ts_sec between ? and ? and ` + removeSentToLocalhostSqlFragment + `
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

	topDomainsByStatus, err := db.Prepare(`
	select
		lm_resolve_domain_mapping(recipient_domain_part) as d, count(lm_resolve_domain_mapping(recipient_domain_part)) as c
	from
		postfix_smtp_message_status
	where
		status = ? and read_ts_sec between ? and ?
	group by
		d collate nocase
	order by
		c desc, d collate nocase asc
	limit 20`)

	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	defer func() {
		if err != nil {
			errorutil.MustSucceed(topDomainsByStatus.Close(), "Closing topDomainsByStatus")
		}
	}()

	topBusiestDomains, err := db.Prepare(`
	select
		lm_resolve_domain_mapping(recipient_domain_part) as d, count(lm_resolve_domain_mapping(recipient_domain_part)) as c
	from
		postfix_smtp_message_status
	where
		read_ts_sec between ? and ? and ` + removeSentToLocalhostSqlFragment + `
	group by
		d collate nocase
	order by
		c desc, d collate nocase asc
	limit 20`)

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
	return listDomainAndCount(ctx, d.queries.topBusiestDomains, interval.From.Unix(), interval.To.Unix())
}

func (d sqlDashboard) TopBouncedDomains(ctx context.Context, interval data.TimeInterval) (Pairs, error) {
	return listDomainAndCount(ctx, d.queries.topDomainsByStatus, parser.BouncedStatus, interval.From.Unix(), interval.To.Unix())
}

func (d sqlDashboard) TopDeferredDomains(ctx context.Context, interval data.TimeInterval) (Pairs, error) {
	return listDomainAndCount(ctx, d.queries.topDomainsByStatus, parser.DeferredStatus, interval.From.Unix(), interval.To.Unix())
}

func (d sqlDashboard) DeliveryStatus(ctx context.Context, interval data.TimeInterval) (Pairs, error) {
	return deliveryStatus(ctx, d.queries.deliveryStatus, interval)
}

func countByStatus(ctx context.Context, stmt *sql.Stmt, status parser.SmtpStatus, interval data.TimeInterval) (int, error) {
	countValue := 0

	if err := stmt.QueryRowContext(ctx, status, interval.From.Unix(), interval.To.Unix()).Scan(&countValue); err != nil {
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
