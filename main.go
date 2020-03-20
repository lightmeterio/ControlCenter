package main

import (
	"bufio"
	"database/sql"
	"encoding/json"
	_ "github.com/mattn/go-sqlite3"
	parser "gitlab.com/lightmeter/postfix-log-parser"
	"gitlab.com/lightmeter/postfix-log-parser/rawparser"
	"log"
	"net/http"
	"os"
)

type Publisher interface {
	Publish(parser.Record)
	Close()
}

type ChannelBasedPublisher struct {
	channel chan parser.Record
}

func (pub *ChannelBasedPublisher) Publish(status parser.Record) {
	pub.channel <- status
}

func (pub *ChannelBasedPublisher) Close() {
	close(pub.channel)
}

func fillDatabase(db *sql.DB, c chan parser.Record) {
	stmt, err := db.Prepare("insert into smtp(recipient_local_part, recipient_domain_part, relay_name, status) values(?, ?, ?, ?)")

	if err != nil {
		log.Fatal("error preparing insert statement")
	}

	for r := range c {
		status, cast := r.Payload.(parser.SmtpSentStatus)

		if !cast {
			continue
		}

		_, err := stmt.Exec(status.RecipientLocalPart, status.RecipientDomainPart, status.RelayName, status.Status)

		if err != nil {
			log.Fatal("error inserting value")
		}
	}
}

func main() {
	c := make(chan parser.Record, 10)
	pub := ChannelBasedPublisher{c}
	go parseLogsFromStdin(&pub)

	os.Remove("lm.db")

	db, err := sql.Open("sqlite3", "lm.db")

	if err != nil {
		log.Fatal("error opening database")
	}

	defer db.Close()

	if _, err := db.Exec("create table smtp(recipient_local_part text, recipient_domain_part text, relay_name text, status text)"); err != nil {
		log.Fatal("error creating database: ", err)
	}

	go fillDatabase(db, c)

	countByStatus := func(status parser.SmtpStatus) int {
		stmt, err := db.Prepare(`select count(status) from smtp where status = ?`)

		if err != nil {
			log.Fatal("error preparing query")
		}

		sentResult, err := stmt.Query(status)

		if err != nil {
			log.Fatal("error querying")
		}

		var countValue int

		sentResult.Next()

		sentResult.Scan(&countValue)

		return countValue
	}

	type domainAndCount struct {
		Domain string
		Count  int
	}

	listDomainAndCount := func(queryStr string, args ...interface{}) []domainAndCount {
		var r []domainAndCount

		stmt, err := db.Prepare(queryStr)

		if err != nil {
			log.Fatal("Error preparing query")
		}

		query, err := stmt.Query(args...)

		if err != nil {
			log.Fatal("Error query")
		}

		for query.Next() {
			var domain string
			var countValue int

			query.Scan(&domain, &countValue)

			r = append(r, domainAndCount{domain, countValue})
		}

		return r
	}

	type deliveryValue struct {
		Status string
		Value  float64
	}

	deliveryStatus := func() []deliveryValue {
		var r []deliveryValue

		query, err := db.Query(`select status, cast(count(status) as float) / cast((select count(status) from smtp) as float) from smtp group by status`)

		if err != nil {
			log.Fatal("Error query")
		}

		for query.Next() {
			var status parser.SmtpStatus
			var value float64

			query.Scan(&status, &value)

			r = append(r, deliveryValue{status.HumanForm(), value * 100})
		}

		return r
	}

	serveJson := func(w http.ResponseWriter, r *http.Request, v interface{}) {
		w.Header().Set("Content-Type", "application/json")
		encoded, _ := json.Marshal(v)
		w.Write(encoded)
	}

	http.HandleFunc("/countByStatus", func(w http.ResponseWriter, r *http.Request) {
		serveJson(w, r, map[string]int{"sent": countByStatus(parser.SentStatus), "deferred": countByStatus(parser.DeferredStatus), "bounced": countByStatus(parser.BouncedStatus)})
	})

	http.HandleFunc("/topBusiestDomains", func(w http.ResponseWriter, r *http.Request) {
		serveJson(w, r, listDomainAndCount(`select recipient_domain_part, count(recipient_domain_part) as c from smtp group by recipient_domain_part order by c desc limit 20`))
	})

	http.HandleFunc("/topBouncedDomains", func(w http.ResponseWriter, r *http.Request) {
		query := `select recipient_domain_part, count(recipient_domain_part) as c from smtp where status = ? and relay_name != "" group by recipient_domain_part order by c desc limit 20`
		serveJson(w, r, listDomainAndCount(query, parser.BouncedStatus.String()))
	})

	http.HandleFunc("/topDeferredDomains", func(w http.ResponseWriter, r *http.Request) {
		query := `select relay_name, count(relay_name) as c from smtp where status = ? and relay_name != "" group by relay_name order by c desc limit 20`
		serveJson(w, r, listDomainAndCount(query, parser.DeferredStatus.String()))
	})

	http.HandleFunc("/deliveryStatus", func(w http.ResponseWriter, r *http.Request) {
		serveJson(w, r, deliveryStatus())
	})

	http.Handle("/", http.FileServer(http.Dir("./static")))

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func parseLogsFromStdin(publisher Publisher) {
	scanner := bufio.NewScanner(os.Stdin)

	for {
		if !scanner.Scan() {
			break
		}

		logLine := scanner.Bytes()

		r, err := parser.Parse(logLine)

		if err != nil {
			if err == rawparser.InvalidHeaderLineError {
				log.Printf("Invalid Postfix header: \"%s\"", string(logLine))
			}
			continue
		}

		publisher.Publish(r)
	}

	publisher.Close()
}
