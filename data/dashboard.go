package data

import (
	"database/sql"
	"gitlab.com/lightmeter/postfix-log-parser"
	"log"
)

type Dashboard struct {
	db *sql.DB
}

func (d *Dashboard) CountByStatus(status parser.SmtpStatus, interval TimeInterval) int {
	return countByStatus(d.db, status, interval)
}

func (d *Dashboard) TopBusiestDomains(interval TimeInterval) []DomainNameAndCount {
	query := `
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
	limit 20`

	return listDomainAndCount(d.db, query, interval.From.Unix(), interval.To.Unix())
}

func (d *Dashboard) TopBouncedDomains(interval TimeInterval) []DomainNameAndCount {
	query := `
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
	limit 20`
	return listDomainAndCount(d.db, query, parser.BouncedStatus, interval.From.Unix(), interval.To.Unix())
}

func (d *Dashboard) TopDeferredDomains(interval TimeInterval) []DomainNameAndCount {
	query := `
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
	limit 20`
	return listDomainAndCount(d.db, query, parser.DeferredStatus, interval.From.Unix(), interval.To.Unix())
}

func (d *Dashboard) DeliveryStatus(interval TimeInterval) []DeliveryValue {
	return deliveryStatus(d.db, interval)
}

func countByStatus(db *sql.DB, status parser.SmtpStatus, interval TimeInterval) int {
	queryStr := `
	select
		count(*)
	from
		postfix_smtp_message_status
	where
		status = ? and read_ts_sec between ? and ?`

	stmt, err := db.Prepare(queryStr)

	if err != nil {
		log.Fatal("error preparing query", err)
	}

	defer stmt.Close()

	query, err := stmt.Query(status, interval.From.Unix(), interval.To.Unix())

	if err != nil {
		log.Fatal("error querying", err)
	}

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

func listDomainAndCount(db *sql.DB, queryStr string, args ...interface{}) []DomainNameAndCount {
	var r []DomainNameAndCount

	stmt, err := db.Prepare(queryStr)

	if err != nil {
		log.Fatal("Error preparing query", err)
	}

	defer stmt.Close()

	query, err := stmt.Query(args...)

	if err != nil {
		log.Fatal("Query error:", err)
	}

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

func deliveryStatus(db *sql.DB, interval TimeInterval) []DeliveryValue {
	var r []DeliveryValue

	stmt, err := db.Prepare(`
	select
		status, count(status)
	from
		postfix_smtp_message_status
	where
		read_ts_sec between ? and ?
	group by status`)

	if err != nil {
		log.Fatal("Prepare error (deliveryStatus):", err)
	}

	defer stmt.Close()

	query, err := stmt.Query(interval.From.Unix(), interval.To.Unix())

	if err != nil {
		log.Fatal("Query error:", err)
	}

	defer query.Close()

	for query.Next() {
		var status parser.SmtpStatus
		var value float64

		query.Scan(&status, &value)

		r = append(r, DeliveryValue{status.String(), value})
	}

	return r
}
