package dashboard

import (
	"database/sql"
	"errors"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"strings"

	"gitlab.com/lightmeter/controlcenter/data"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	parser "gitlab.com/lightmeter/postfix-log-parser"
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

	CountByStatus(status parser.SmtpStatus, interval data.TimeInterval) int
	TopBusiestDomains(interval data.TimeInterval) Pairs
	TopBouncedDomains(interval data.TimeInterval) Pairs
	TopDeferredDomains(interval data.TimeInterval) Pairs
	DeliveryStatus(interval data.TimeInterval) Pairs
}

type SqlDbDashboard struct {
	queries queries
}

const removeSentToLocalhostSqlFragment = `((process_ip is not null and relay_ip != process_ip) or (process_ip is null and relay_name != "127.0.0.1"))`

func New(db dbconn.RoConn) (SqlDbDashboard, error) {
	countByStatus, err := db.Prepare(`
	select
		count(*)
	from
		postfix_smtp_message_status
	where
		status = ? and read_ts_sec between ? and ? and ` + removeSentToLocalhostSqlFragment)

	if err != nil {
		return SqlDbDashboard{}, errorutil.WrapError(err)
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
		return SqlDbDashboard{}, errorutil.WrapError(err)
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
		return SqlDbDashboard{}, errorutil.WrapError(err)
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
		return SqlDbDashboard{}, errorutil.WrapError(err)
	}

	defer func() {
		if err != nil {
			errorutil.MustSucceed(topBusiestDomains.Close(), "Closing topBusiestDomains")
		}
	}()

	return SqlDbDashboard{
		queries: queries{
			countByStatus:      countByStatus,
			deliveryStatus:     deliveryStatus,
			topBusiestDomains:  topBusiestDomains,
			topDomainsByStatus: topDomainsByStatus,
		},
	}, nil
}

var ErrClosingDashboardQueries = errors.New("Error closing any of the dashboard queries!")

func (d SqlDbDashboard) Close() error {
	errCountByStatus := d.queries.countByStatus.Close()
	errDeliveryStatus := d.queries.deliveryStatus.Close()
	errTopBusiestDomains := d.queries.topBusiestDomains.Close()
	errTopBouncedDomains := d.queries.topDomainsByStatus.Close()

	if errCountByStatus != nil ||
		errDeliveryStatus != nil ||
		errTopBusiestDomains != nil ||
		errTopBouncedDomains != nil {

		return errorutil.WrapError(ErrClosingDashboardQueries)
	}

	return nil
}

func (d SqlDbDashboard) CountByStatus(status parser.SmtpStatus, interval data.TimeInterval) int {
	return countByStatus(d.queries.countByStatus, status, interval)
}

func (d SqlDbDashboard) TopBusiestDomains(interval data.TimeInterval) Pairs {
	return listDomainAndCount(d.queries.topBusiestDomains, interval.From.Unix(), interval.To.Unix())
}

func (d SqlDbDashboard) TopBouncedDomains(interval data.TimeInterval) Pairs {
	return listDomainAndCount(d.queries.topDomainsByStatus, parser.BouncedStatus, interval.From.Unix(), interval.To.Unix())
}

func (d SqlDbDashboard) TopDeferredDomains(interval data.TimeInterval) Pairs {
	return listDomainAndCount(d.queries.topDomainsByStatus, parser.DeferredStatus, interval.From.Unix(), interval.To.Unix())
}

func (d SqlDbDashboard) DeliveryStatus(interval data.TimeInterval) Pairs {
	return deliveryStatus(d.queries.deliveryStatus, interval)
}

func countByStatus(stmt *sql.Stmt, status parser.SmtpStatus, interval data.TimeInterval) int {
	countValue := 0
	errorutil.MustSucceed(stmt.QueryRow(status, interval.From.Unix(), interval.To.Unix()).Scan(&countValue), "")
	return countValue
}

// rowserrcheck is buggy and unable to see that the query errors are being checked
// when query.Close() is inside a closure
//nolint:rowserrcheck
func listDomainAndCount(stmt *sql.Stmt, args ...interface{}) Pairs {
	r := Pairs{}

	query, err := stmt.Query(args...)

	errorutil.MustSucceed(err, "ListDomainAndCount")

	defer func() { errorutil.MustSucceed(query.Close(), "") }()

	for query.Next() {
		var domain string
		var countValue int

		errorutil.MustSucceed(query.Scan(&domain, &countValue), "scan")

		// If the relay info is not available, use a placeholder
		if len(domain) == 0 {
			domain = "<none>"
		}

		r = append(r, Pair{strings.ToLower(domain), countValue})
	}

	errorutil.MustSucceed(query.Err(), "Error on rows")

	return r
}

// rowserrcheck is buggy and unable to see that the query errors are being checked
// when query.Close() is inside a closure
//nolint:rowserrcheck
func deliveryStatus(stmt *sql.Stmt, interval data.TimeInterval) Pairs {
	r := Pairs{}

	query, err := stmt.Query(interval.From.Unix(), interval.To.Unix())

	errorutil.MustSucceed(err, "DeliveryStatus")

	defer func() { errorutil.MustSucceed(query.Close(), "") }()

	for query.Next() {
		var status parser.SmtpStatus
		var value int

		errorutil.MustSucceed(query.Scan(&status, &value), "scan")

		r = append(r, Pair{status.String(), value})
	}

	errorutil.MustSucceed(query.Err(), "Error on rows")

	return r
}
