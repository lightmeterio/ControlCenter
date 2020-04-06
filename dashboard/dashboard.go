package dashboard

import (
	"database/sql"
	"errors"
	"gitlab.com/lightmeter/controlcenter/data"
	"gitlab.com/lightmeter/controlcenter/util"
	"gitlab.com/lightmeter/postfix-log-parser"
)

type queries struct {
	countByStatus      *sql.Stmt
	deliveryStatus     *sql.Stmt
	topBusiestDomains  *sql.Stmt
	topDeferredDomains *sql.Stmt
	topBouncedDomains  *sql.Stmt
}

type Dashboard struct {
	queries queries
}

func New(db *sql.DB) (Dashboard, error) {
	countByStatus, err := db.Prepare(`
	select
		count(*)
	from
		postfix_smtp_message_status
	where
		status = ? and read_ts_sec between ? and ?`)

	if err != nil {
		return Dashboard{}, err
	}

	defer func() {
		if err != nil {
			util.MustSucceed(countByStatus.Close(), "Closing countByStatus")
		}
	}()

	deliveryStatus, err := db.Prepare(`
	select
		status, count(status)
	from
		postfix_smtp_message_status
	where
		read_ts_sec between ? and ?
	group by status`)

	if err != nil {
		return Dashboard{}, err
	}

	defer func() {
		if err != nil {
			util.MustSucceed(deliveryStatus.Close(), "Closing deliveryStatus")
		}
	}()

	topDeferredDomains, err := db.Prepare(`
	select
		relay_name, count(relay_name) as c
	from
		postfix_smtp_message_status
	where
		status = ? and read_ts_sec between ? and ?
	group by
		relay_name
	order by
		c desc
	limit 20`)

	if err != nil {
		return Dashboard{}, err
	}

	defer func() {
		if err != nil {
			util.MustSucceed(topDeferredDomains.Close(), "Closing topDeferredDomains")
		}
	}()

	topBouncedDomains, err := db.Prepare(`
	select
		recipient_domain_part, count(recipient_domain_part) as c
	from
		postfix_smtp_message_status
	where
		status = ? and read_ts_sec between ? and ?
	group by
		recipient_domain_part
	order by
		c desc
	limit 20`)

	if err != nil {
		return Dashboard{}, err
	}

	defer func() {
		if err != nil {
			util.MustSucceed(topBouncedDomains.Close(), "Closing topBouncedDomains")
		}
	}()

	topBusiestDomains, err := db.Prepare(`
	select
		recipient_domain_part, count(recipient_domain_part) as c
	from
		postfix_smtp_message_status
	where
			read_ts_sec between ? and ?
	group by
		recipient_domain_part 
	order by
		c desc
	limit 20`)

	if err != nil {
		return Dashboard{}, err
	}

	defer func() {
		if err != nil {
			util.MustSucceed(topBusiestDomains.Close(), "Closing topBusiestDomains")
		}
	}()

	return Dashboard{
		queries: queries{
			countByStatus:      countByStatus,
			deliveryStatus:     deliveryStatus,
			topBusiestDomains:  topBusiestDomains,
			topDeferredDomains: topDeferredDomains,
			topBouncedDomains:  topBouncedDomains,
		},
	}, nil
}

func (d Dashboard) Close() error {
	errCountByStatus := d.queries.countByStatus.Close()
	errDeliveryStatus := d.queries.deliveryStatus.Close()
	errTopBusiestDomains := d.queries.topBusiestDomains.Close()
	errTopDeferredDomains := d.queries.topDeferredDomains.Close()
	errTopBouncedDomains := d.queries.topBouncedDomains.Close()

	if errCountByStatus != nil ||
		errDeliveryStatus != nil ||
		errTopBusiestDomains != nil ||
		errTopDeferredDomains != nil ||
		errTopBouncedDomains != nil {

		return errors.New("Error closing any of the dashboard queries!")
	}

	return nil
}

func (d Dashboard) CountByStatus(status parser.SmtpStatus, interval data.TimeInterval) int {
	return countByStatus(d.queries.countByStatus, status, interval)
}

func (d Dashboard) TopBusiestDomains(interval data.TimeInterval) []DomainNameAndCount {
	return listDomainAndCount(d.queries.topBusiestDomains, interval.From.Unix(), interval.To.Unix())
}

func (d Dashboard) TopBouncedDomains(interval data.TimeInterval) []DomainNameAndCount {
	return listDomainAndCount(d.queries.topBouncedDomains, parser.BouncedStatus, interval.From.Unix(), interval.To.Unix())
}

func (d Dashboard) TopDeferredDomains(interval data.TimeInterval) []DomainNameAndCount {
	return listDomainAndCount(d.queries.topDeferredDomains, parser.DeferredStatus, interval.From.Unix(), interval.To.Unix())
}

func (d Dashboard) DeliveryStatus(interval data.TimeInterval) []DeliveryValue {
	return deliveryStatus(d.queries.deliveryStatus, interval)
}

func countByStatus(stmt *sql.Stmt, status parser.SmtpStatus, interval data.TimeInterval) int {
	query, err := stmt.Query(status, interval.From.Unix(), interval.To.Unix())

	util.MustSucceed(err, "CountByStatus")

	defer query.Close()

	var countValue int

	query.Next()

	query.Scan(&countValue)

	return countValue
}

type DomainNameAndCount struct {
	Domain string
	Count  int
}

func listDomainAndCount(stmt *sql.Stmt, args ...interface{}) []DomainNameAndCount {
	var r []DomainNameAndCount

	query, err := stmt.Query(args...)

	util.MustSucceed(err, "ListDomainAndCount")

	defer query.Close()

	for query.Next() {
		var domain string
		var countValue int

		query.Scan(&domain, &countValue)

		// If the relay info is not available, use a placeholder
		if len(domain) == 0 {
			domain = "<none>"
		}

		r = append(r, DomainNameAndCount{domain, countValue})
	}

	return r
}

type DeliveryValue struct {
	Status string
	Value  float64
}

func deliveryStatus(stmt *sql.Stmt, interval data.TimeInterval) []DeliveryValue {
	var r []DeliveryValue

	query, err := stmt.Query(interval.From.Unix(), interval.To.Unix())

	util.MustSucceed(err, "DeliveryStatus")

	defer query.Close()

	for query.Next() {
		var status parser.SmtpStatus
		var value float64

		query.Scan(&status, &value)

		r = append(r, DeliveryValue{status.String(), value})
	}

	return r
}
