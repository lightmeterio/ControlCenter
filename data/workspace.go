package data

import (
	"database/sql"
	"errors"
	_ "github.com/mattn/go-sqlite3"
	"gitlab.com/lightmeter/controlcenter/data/postfix"
	parser "gitlab.com/lightmeter/postfix-log-parser"
	"log"
	"os"
	"path"
	"time"
)

var (
	sqliteTransactionTime time.Duration = 1000 * time.Millisecond
)

type Config struct {
	Location    *time.Location
	DefaultYear int
}

const (
	filename = "data.db"
)

type Workspace struct {
	config Config

	writerConnection *sql.DB

	// connection that is unable to do any changes to the database (like inserts, updates, etc.)
	readerConnection *sql.DB

	dirName string

	records chan Record
}

func NewWorkspace(workspaceDirectory string, config Config) (Workspace, error) {
	if err := os.MkdirAll(workspaceDirectory, os.ModePerm); err != nil {
		return Workspace{}, errors.New("Error creating working directory " + workspaceDirectory + ": " + err.Error())
	}

	dbFilename := path.Join(workspaceDirectory, filename)

	writerConnection, err := sql.Open("sqlite3", dbFilename+`?mode=rwc&cache=shared&_loc=auto`)

	if err != nil {
		return Workspace{}, err
	}

	// TODO: set page size only on the database is created!
	if _, err := writerConnection.Exec(`PRAGMA page_size = 32768`); err != nil {
		writerConnection.Close()
		return Workspace{}, err
	}

	readerConnection, err := sql.Open("sqlite3", dbFilename+`?mode=ro,_query_only=true&cache=shared&_loc=auto`)

	if err != nil {
		writerConnection.Close()
		return Workspace{}, err
	}

	if _, err := writerConnection.Exec(`create table if not exists postfix_smtp_message_status(
		read_ts_sec           integer,
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
		readerConnection.Close()
		writerConnection.Close()
		return Workspace{}, err
	}

	if _, err := writerConnection.Exec(`create index if not exists time_index
		on postfix_smtp_message_status (read_ts_sec)`); err != nil {
		readerConnection.Close()
		writerConnection.Close()
		return Workspace{}, err
	}

	return Workspace{
		config:           config,
		writerConnection: writerConnection,
		readerConnection: readerConnection,
		dirName:          workspaceDirectory,
		records:          make(chan Record, 10),
	}, nil
}

func (ws *Workspace) Dashboard() Dashboard {
	return Dashboard{db: ws.readerConnection}
}

func (ws *Workspace) NewPublisher() Publisher {
	return &ChannelBasedPublisher{ws.records}
}

func (ws *Workspace) Run() <-chan interface{} {
	done := make(chan interface{})
	converter := postfix.NewTimeConverter(buildInitialTime(ws.readerConnection, ws.config.DefaultYear, ws.config.Location))
	go fillDatabase(converter, ws.writerConnection, ws.records, done)
	return done
}

func (ws *Workspace) Close() error {
	errReader := ws.readerConnection.Close()
	errWriter := ws.writerConnection.Close()

	if errWriter != nil || errReader != nil {
		return errors.New("error closing database: writer:(" + errWriter.Error() + "), reader: (" + errReader.Error() + ")")
	}

	return nil
}

func (ws *Workspace) HasLogs() bool {
	q, err := ws.readerConnection.Query(`select count(*) from postfix_smtp_message_status`)

	if err != nil {
		log.Fatal("Error checking if database has logs:", err)
	}

	defer q.Close()

	if !q.Next() {
		return false
	}

	var value int

	if q.Scan(&value) != nil {
		return false
	}

	return value > 0
}

func fillDatabase(timeConverter postfix.TimeConverter, db *sql.DB, c <-chan Record, done chan<- interface{}) {
	insertQuery := `insert into postfix_smtp_message_status(
		read_ts_sec,
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
	) values(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	insertCb := func(stmt *sql.Stmt, r Record) bool {
		ts := timeConverter.Convert(r.Header.Time)

		status, cast := r.Payload.(parser.SmtpSentStatus)

		if !cast {
			return false
		}

		_, err := stmt.Exec(
			ts,
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

	var tx *sql.Tx = nil
	var stmt *sql.Stmt = nil
	var err error = nil

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
		if err := stmt.Close(); err != nil {
			log.Fatal("Error closing insert statement:", err)
		}

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

	timeoutTimer := time.Tick(timeout)

	startTransaction()

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

			break

		case <-timeoutTimer:
			restartTransaction()
		}
	}
}

func buildInitialTime(db *sql.DB, logYear int, timezone *time.Location) (parser.Time, int, *time.Location) {
	// FIXME: this query is way too complicated for something so simple
	q, err := db.Query(`select read_ts_sec from postfix_smtp_message_status where rowid = (select max(rowid) from postfix_smtp_message_status)`)
	if err != nil {
		log.Fatal("Error getting time for the last element:", err)
	}

	defer q.Close()

	if !q.Next() {
		log.Println("Could not obtain time from existing database, using defaulted one:", logYear)
		return parser.Time{}, logYear, timezone
	}

	var v int64
	if err := q.Scan(&v); err != nil {
		log.Println("Could not obtain time from existing database, using defaulted one")
		return parser.Time{}, logYear, timezone
	}

	ts := time.Unix(v, 0).In(timezone)

	log.Println("Using initial time as:", ts)

	return parser.Time{
		Month:  ts.Month(),
		Day:    uint8(ts.Day()),
		Hour:   uint8(ts.Hour()),
		Minute: uint8(ts.Minute()),
		Second: uint8(ts.Second()),
	}, ts.Year(), timezone
}
