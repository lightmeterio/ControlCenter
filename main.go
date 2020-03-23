//go:generate go run ./assets/assets_generate.go

package main

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"github.com/hpcloud/tail"
	_ "github.com/mattn/go-sqlite3"
	parser "gitlab.com/lightmeter/postfix-log-parser"
	"gitlab.com/lightmeter/postfix-log-parser/rawparser"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
)

type watchableFilenames []string

func (this watchableFilenames) String() string {
	return strings.Join(this, ", ")
}

func (this *watchableFilenames) Set(value string) error {
	*this = append(*this, value)
	return nil
}

var (
	filesToWatch       watchableFilenames
	watchFromStdin     bool
	workspaceDirectory string
)

func init() {
	flag.Var(&filesToWatch, "watch", "File to watch (can be used multiple times")
	flag.BoolVar(&watchFromStdin, "stdin", false, "Read log lines from stdin")
	flag.StringVar(&workspaceDirectory, "workspace", "lm_data", "Path to an existing directory to store all working data")
}

type Publisher interface {
	Publish(parser.Record)
	Close()
}

type ChannelBasedPublisher struct {
	channel chan<- parser.Record
}

func (pub *ChannelBasedPublisher) Publish(status parser.Record) {
	pub.channel <- status
}

func (pub *ChannelBasedPublisher) Close() {
	close(pub.channel)
}

func fillDatabase(db *sql.DB, c chan parser.Record) {
	stmt, err := db.Prepare(`insert into smtp(
			queue,
			recipient_local_part,
			recipient_domain_part,
			relay_name,
			relay_ip,
			relay_port,
			delay,
			delay_smtpd,
			delay_cleanup,
			delay_qmgr,
			delay_smtp,
			dsn,
			status
	) values(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)

	if err != nil {
		log.Fatal("error preparing insert statement")
	}

	for r := range c {
		status, cast := r.Payload.(parser.SmtpSentStatus)

		if !cast {
			continue
		}

		_, err := stmt.Exec(
			status.Queue,
			status.RecipientLocalPart,
			status.RecipientDomainPart,
			status.RelayName,
			[]byte(status.RelayIP),
			status.RelayPort,
			status.Delay,
			status.Delays.Smtpd,
			status.Delays.Cleanup,
			status.Delays.Qmgr,
			status.Delays.Smtp,
			status.Dsn,
			status.Status)

		if err != nil {
			log.Fatal("error inserting value")
		}
	}
}

func countByStatus(db *sql.DB, status parser.SmtpStatus) int {
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

func listDomainAndCount(db *sql.DB, queryStr string, args ...interface{}) []domainAndCount {
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

func deliveryStatus(db *sql.DB) []deliveryValue {
	var r []deliveryValue

	query, err := db.Query(`select status, count(status) from smtp group by status`)

	if err != nil {
		log.Fatal("Error query")
	}

	for query.Next() {
		var status parser.SmtpStatus
		var value float64

		query.Scan(&status, &value)

		r = append(r, deliveryValue{status.String(), value})
	}

	return r
}

func main() {
	flag.Parse()

	if err := os.MkdirAll(workspaceDirectory, os.ModePerm); err != nil {
		log.Fatal("Error creating working directory", workspaceDirectory, ":", err)
	}

	dbFilename := path.Join(workspaceDirectory, "logs.db")

	watchLocation, err := func() (*tail.SeekInfo, error) {
		s, err := os.Stat(dbFilename)

		if os.IsNotExist(err) {
			return &tail.SeekInfo{
				Offset: 0,
				Whence: os.SEEK_SET,
			}, nil
		}

		if s.IsDir() {
			return nil, errors.New(dbFilename + " must be a regular file!")
		}

		return nil, nil
	}()

	if err != nil {
		log.Fatal("Setup error:", err)
	}

	db, err := sql.Open("sqlite3", dbFilename)

	if err != nil {
		log.Fatal("error opening database")
	}

	defer db.Close()

	if _, err := db.Exec(`create table if not exists smtp(
			queue                 blob,
			recipient_local_part  text,
			recipient_domain_part text,
			relay_name            text,
			relay_ip              blob,
			relay_port            uint16,
			delay                 double,
			delay_smtpd   				double,
			delay_cleanup 				double,
			delay_qmgr    				double,
			delay_smtp    				double,
			dsn                   text,
			status                integer
		)`); err != nil {

		log.Fatal("error creating database: ", err)
	}

	c := make(chan parser.Record, 10)

	pub := ChannelBasedPublisher{c}

	if watchFromStdin {
		go parseLogsFromStdin(&pub)
	}

	for _, filename := range filesToWatch {
		go watchFileForChanges(filename, watchLocation, &pub)
	}

	go fillDatabase(db, c)

	serveJson := func(w http.ResponseWriter, r *http.Request, v interface{}) {
		w.Header().Set("Content-Type", "application/json")
		encoded, _ := json.Marshal(v)
		w.Write(encoded)
	}

	http.HandleFunc("/api/countByStatus", func(w http.ResponseWriter, r *http.Request) {
		serveJson(w, r, map[string]int{
			"sent":     countByStatus(db, parser.SentStatus),
			"deferred": countByStatus(db, parser.DeferredStatus),
			"bounced":  countByStatus(db, parser.BouncedStatus),
		})
	})

	http.HandleFunc("/api/topBusiestDomains", func(w http.ResponseWriter, r *http.Request) {
		serveJson(w, r, listDomainAndCount(db, `select recipient_domain_part, count(recipient_domain_part) as c from smtp group by recipient_domain_part order by c desc limit 20`))
	})

	http.HandleFunc("/api/topBouncedDomains", func(w http.ResponseWriter, r *http.Request) {
		query := `select recipient_domain_part, count(recipient_domain_part) as c from smtp where status = ? and relay_name != "" group by recipient_domain_part order by c desc limit 20`
		serveJson(w, r, listDomainAndCount(db, query, parser.BouncedStatus))
	})

	http.HandleFunc("/api/topDeferredDomains", func(w http.ResponseWriter, r *http.Request) {
		query := `select relay_name, count(relay_name) as c from smtp where status = ? and relay_name != "" group by relay_name order by c desc limit 20`
		serveJson(w, r, listDomainAndCount(db, query, parser.DeferredStatus))
	})

	http.HandleFunc("/api/deliveryStatus", func(w http.ResponseWriter, r *http.Request) {
		serveJson(w, r, deliveryStatus(db))
	})

	http.Handle("/", http.FileServer(httpAssets))

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func tryToParseAndPublish(line []byte, publisher Publisher) {
	r, err := parser.Parse(line)

	if err != nil {
		if err == rawparser.InvalidHeaderLineError {
			log.Printf("Invalid Postfix header: \"%s\"", string(line))
		}
		return
	}

	publisher.Publish(r)
}

func watchFileForChanges(filename string, location *tail.SeekInfo, publisher Publisher) error {
	log.Println("Now watching file", filename, "for changes from the", func() string {
		if location == nil {
			return "end"
		}

		return "beginning"
	}())

	t, err := tail.TailFile(filename, tail.Config{
		Follow:   true,
		ReOpen:   true,
		Logger:   tail.DiscardingLogger,
		Location: location,
	})

	if err != nil {
		return err
	}

	for line := range t.Lines {
		tryToParseAndPublish([]byte(line.Text), publisher)
	}

	return nil
}

func parseLogsFromStdin(publisher Publisher) {
	scanner := bufio.NewScanner(os.Stdin)

	for {
		if !scanner.Scan() {
			break
		}

		tryToParseAndPublish(scanner.Bytes(), publisher)
	}

	publisher.Close()
}
