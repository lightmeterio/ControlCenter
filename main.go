package main

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"github.com/hpcloud/tail"
	_ "github.com/mattn/go-sqlite3"
	"gitlab.com/lightmeter/controlcenter/staticdata"
	parser "gitlab.com/lightmeter/postfix-log-parser"
	"gitlab.com/lightmeter/postfix-log-parser/rawparser"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"time"
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
	filesToWatch          watchableFilenames
	watchFromStdin        bool
	workspaceDirectory    string
	importOnly            bool
	sqliteTransactionTime time.Duration    = 1000 * time.Millisecond
	getNow                func() time.Time = time.Now
)

func init() {
	flag.Var(&filesToWatch, "watch", "File to watch (can be used multiple times")
	flag.BoolVar(&watchFromStdin, "stdin", false, "Read log lines from stdin")
	flag.StringVar(&workspaceDirectory, "workspace", "lightmeter_workspace", "Path to an existing directory to store all working data")
	flag.BoolVar(&importOnly, "importonly", false,
		"Only import logs from stdin, exiting imediately, without running the full application. Implies -stdin")
}

type Record struct {
	Header  parser.Header
	Payload parser.Payload
	Time    time.Time
}

type Publisher interface {
	Publish(Record)
	Close()
}

type ChannelBasedPublisher struct {
	channel chan<- Record
}

func (pub *ChannelBasedPublisher) Publish(status Record) {
	pub.channel <- status
}

func (pub *ChannelBasedPublisher) Close() {
	close(pub.channel)
}

func fillDatabase(db *sql.DB, c <-chan Record, done chan<- bool) {
	insertQuery := `insert into postfix_smtp_message_status(
			read_ts_sec,
			read_ts_nsec,
			time_month,
			time_day,
			time_hour,
			time_minute,
			time_second,
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
	) values(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	insertCb := func(stmt *sql.Stmt, r Record) bool {
		status, cast := r.Payload.(parser.SmtpSentStatus)

		if !cast {
			return false
		}

		_, err := stmt.Exec(
			r.Time.Unix(),
			r.Time.UnixNano(),
			r.Header.Time.Month,
			r.Header.Time.Day,
			r.Header.Time.Hour,
			r.Header.Time.Minute,
			r.Header.Time.Second,
			status.Queue,
			status.RecipientLocalPart,
			status.RecipientDomainPart,
			status.RelayName,
			status.RelayIP,
			status.RelayPort,
			status.Delay,
			status.Delays.Smtpd,
			status.Delays.Cleanup,
			status.Delays.Qmgr,
			status.Delays.Smtp,
			status.Dsn,
			status.Status)

		if err != nil {
			log.Fatal("error inserting value:", err)
		}

		return true
	}

	performInsertsIntoDbGroupingInTransactions(db, c, sqliteTransactionTime, insertQuery, insertCb)

	done <- true
}

func performInsertsIntoDbGroupingInTransactions(db *sql.DB,
	c <-chan Record, timeout time.Duration,
	insertQuery string,
	insertCb func(*sql.Stmt, Record) bool) {

	var tx *sql.Tx
	var stmt *sql.Stmt
	var err error

	countPerTransaction := 0

	startTransaction := func() {
		tx, err = db.Begin()

		if err != nil {
			log.Fatal(`Error preparing transaction:`, err)
		}

		stmt, err = tx.Prepare(insertQuery)

		if err != nil {
			log.Fatal("error preparing insert statement:", err)
		}
	}

	closeTransaction := func() {
		if countPerTransaction == 0 {
			if err := tx.Rollback(); err != nil {
				log.Fatal("Error discarding empty trasaction", err)
			}

			return
		}

		// NOTE: improve it to be used for benchmarking
		log.Println("Inserted", countPerTransaction, "rows in a transaction")

		countPerTransaction = 0

		if err := tx.Commit(); err != nil {
			log.Fatal("Error commiting transaction:", err)
		}
	}

	restartTransaction := func() {
		closeTransaction()
		startTransaction()
	}

	startTransaction()

	timeoutTimer := time.Tick(timeout)

	// TODO: improve this by start a transaction only if there are new stuff to be inserted
	// in the database
	for {
		select {
		case r, ok := <-c:
			if !ok {
				closeTransaction()
				return
			}

			if insertCb(stmt, r) {
				countPerTransaction += 1
			}

		case <-timeoutTimer:
			restartTransaction()
		}
	}
}

func countByStatus(db *sql.DB, status parser.SmtpStatus) int {
	stmt, err := db.Prepare(`select count(status) from postfix_smtp_message_status where status = ?`)

	if err != nil {
		log.Fatal("error preparing query", err)
	}

	sentResult, err := stmt.Query(status)

	if err != nil {
		log.Fatal("error querying", err)
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
		log.Fatal("Error preparing query", err)
	}

	query, err := stmt.Query(args...)

	if err != nil {
		log.Fatal("Query error:", err)
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

	query, err := db.Query(`select status, count(status) from postfix_smtp_message_status group by status`)

	if err != nil {
		log.Fatal("Query error:", err)
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

	dbFilename := path.Join(workspaceDirectory, "data.db")

	//isCreatingNewDb := func() bool {
	//	if _, err := os.Stat(dbFilename); os.IsNotExist(err) {
	//		return true
	//	}

	//	return false
	//}()

	logFilesWatchLocation, err := func() (*tail.SeekInfo, error) {
		s, err := os.Stat(dbFilename)

		// in case the database does not yet exist, we watch the log files
		// from their beginning, as an "first execution" process,
		// to import as much as we can into the database.
		if os.IsNotExist(err) {
			return &tail.SeekInfo{
				Offset: 0,
				Whence: os.SEEK_SET,
			}, nil
		}

		if s.IsDir() {
			return nil, errors.New(dbFilename + " must be a regular file!")
		}

		return &tail.SeekInfo{
			Offset: 0,
			Whence: os.SEEK_END,
		}, nil
	}()

	if err != nil {
		log.Fatal("Setup error:", err)
	}

	writerConnection, err := sql.Open("sqlite3", dbFilename+`?cache=shared&_loc=auto`)

	if err != nil {
		log.Fatal("error opening write connection to the database", err)
	}

	// TODO: set page size only on the database is created!
	if _, err := writerConnection.Exec(`PRAGMA page_size = 32768`); err != nil {
		log.Fatal("error setting page_size:", err)
	}

	readerConnection, err := sql.Open("sqlite3", dbFilename+`?_query_only=true&cache=shared&_loc=auto`)

	if err != nil {
		log.Fatal("error opening read connection to the database", err)
	}

	defer writerConnection.Close()
	defer readerConnection.Close()

	if _, err := writerConnection.Exec(`create table if not exists postfix_smtp_message_status(
			read_ts_sec           integer,
			read_ts_nsec          integer,
			time_month            integer,
			time_day              integer,
			time_hour             integer,
			time_minute           integer,
			time_second           integer,
			queue                 string,
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

	c := make(chan Record, 100)

	pub := ChannelBasedPublisher{c}

	doneWithDatabase := make(chan bool)

	go fillDatabase(writerConnection, c, doneWithDatabase)

	if importOnly {
		parseLogsFromStdin(&pub)
		<-doneWithDatabase
		return
	}

	if watchFromStdin {
		go parseLogsFromStdin(&pub)
	}

	for _, filename := range filesToWatch {
		go watchFileForChanges(filename, logFilesWatchLocation, &pub)
	}

	serveJson := func(w http.ResponseWriter, r *http.Request, v interface{}) {
		w.Header().Set("Content-Type", "application/json")
		encoded, _ := json.Marshal(v)
		w.Write(encoded)
	}

	http.HandleFunc("/api/countByStatus", func(w http.ResponseWriter, r *http.Request) {
		serveJson(w, r, map[string]int{
			"sent":     countByStatus(readerConnection, parser.SentStatus),
			"deferred": countByStatus(readerConnection, parser.DeferredStatus),
			"bounced":  countByStatus(readerConnection, parser.BouncedStatus),
		})
	})

	http.HandleFunc("/api/topBusiestDomains", func(w http.ResponseWriter, r *http.Request) {
		serveJson(w, r, listDomainAndCount(readerConnection, `select recipient_domain_part, count(recipient_domain_part) as c from postfix_smtp_message_status group by recipient_domain_part order by c desc limit 20`))
	})

	http.HandleFunc("/api/topBouncedDomains", func(w http.ResponseWriter, r *http.Request) {
		query := `select recipient_domain_part, count(recipient_domain_part) as c from postfix_smtp_message_status where status = ? and relay_name != "" group by recipient_domain_part order by c desc limit 20`
		serveJson(w, r, listDomainAndCount(readerConnection, query, parser.BouncedStatus))
	})

	http.HandleFunc("/api/topDeferredDomains", func(w http.ResponseWriter, r *http.Request) {
		query := `select relay_name, count(relay_name) as c from postfix_smtp_message_status where status = ? and relay_name != "" group by relay_name order by c desc limit 20`
		serveJson(w, r, listDomainAndCount(readerConnection, query, parser.DeferredStatus))
	})

	http.HandleFunc("/api/deliveryStatus", func(w http.ResponseWriter, r *http.Request) {
		serveJson(w, r, deliveryStatus(readerConnection))
	})

	http.Handle("/", http.FileServer(staticdata.HttpAssets))

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func tryToParseAndPublish(line []byte, now time.Time, publisher Publisher) {
	h, p, err := parser.Parse(line)

	if err != nil {
		if err == rawparser.InvalidHeaderLineError {
			log.Printf("Invalid Postfix header: \"%s\"", string(line))
		}
		return
	}

	publisher.Publish(Record{Time: now, Header: h, Payload: p})
}

func watchFileForChanges(filename string, location *tail.SeekInfo, publisher Publisher) error {
	log.Println("Now watching file", filename, "for changes from the", func() string {
		if location.Whence == os.SEEK_END {
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
		tryToParseAndPublish([]byte(line.Text), line.Time, publisher)
	}

	return nil
}

func parseLogsFromStdin(publisher Publisher) {
	scanner := bufio.NewScanner(os.Stdin)

	for {
		if scanner.Scan() {
			tryToParseAndPublish(scanner.Bytes(), getNow(), publisher)
		}
	}
}
