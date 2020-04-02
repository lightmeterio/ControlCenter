package data

import (
	"database/sql"
	"gitlab.com/lightmeter/postfix-log-parser"
	"log"
)

type Dashboard struct {
	db *sql.DB
}

func (d *Dashboard) CountByStatus(status parser.SmtpStatus) int {
	return countByStatus(d.db, status)
}

func (d *Dashboard) TopBusiestDomains() []DomainNameAndCount {
	return listDomainAndCount(d.db, `select recipient_domain_part, count(recipient_domain_part) as c from postfix_smtp_message_status group by recipient_domain_part order by c desc limit 20`)
}

func (d *Dashboard) TopBouncedDomains() []DomainNameAndCount {
	query := `select recipient_domain_part, count(recipient_domain_part) as c from postfix_smtp_message_status where status = ? and relay_name != "" group by recipient_domain_part order by c desc limit 20`
	return listDomainAndCount(d.db, query, parser.BouncedStatus)
}

func (d *Dashboard) TopDeferredDomains() []DomainNameAndCount {
	query := `select relay_name, count(relay_name) as c from postfix_smtp_message_status where status = ? and relay_name != "" group by relay_name order by c desc limit 20`
	return listDomainAndCount(d.db, query, parser.DeferredStatus)
}

func (d *Dashboard) DeliveryStatus() []DeliveryValue {
	return deliveryStatus(d.db)
}

func countByStatus(db *sql.DB, status parser.SmtpStatus) int {
	stmt, err := db.Prepare(`select count(status) from postfix_smtp_message_status where status = ?`)

	if err != nil {
		log.Fatal("error preparing query", err)
	}

	defer stmt.Close()

	sentResult, err := stmt.Query(status)

	if err != nil {
		log.Fatal("error querying", err)
	}

	defer sentResult.Close()

	var countValue int

	sentResult.Next()

	sentResult.Scan(&countValue)

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

		r = append(r, DomainNameAndCount{domain, countValue})
	}

	return r
}

type DeliveryValue struct {
	Status string
	Value  float64
}

func deliveryStatus(db *sql.DB) []DeliveryValue {
	var r []DeliveryValue

	query, err := db.Query(`select status, count(status) from postfix_smtp_message_status group by status`)

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
