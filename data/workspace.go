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
	filename         = "data.db"
	recordsQueueSize = 100
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

	for _, handler := range payloadHandlers {
		if err := handler.creator(writerConnection); err != nil {
			readerConnection.Close()
			writerConnection.Close()
			return Workspace{}, err
		}
	}

	return Workspace{
		config:           config,
		writerConnection: writerConnection,
		readerConnection: readerConnection,
		dirName:          workspaceDirectory,
		records:          make(chan Record, recordsQueueSize),
	}, nil
}

func (ws *Workspace) Dashboard() Dashboard {
	return Dashboard{db: ws.readerConnection}
}

func (ws *Workspace) NewPublisher() Publisher {
	return &ChannelBasedPublisher{ws.records}
}

func (ws *Workspace) Run() <-chan interface{} {
	doneTimestamping := make(chan interface{})
	doneInsertingOnDatabase := make(chan interface{})

	newYearNotifier := func(year int, old parser.Time, new parser.Time) {
		log.Println("Bumping year", year, ", old:", old, ", new:", new)
	}

	time, year := buildInitialTime(ws.readerConnection, ws.config.DefaultYear, ws.config.Location)

	converter := postfix.NewTimeConverter(time, year, ws.config.Location, newYearNotifier)
	timedRecords := make(chan TimedRecord, recordsQueueSize)

	go stampLogsWithTimeAndWaitUntilDatabaseIsFinished(converter, ws.records, timedRecords, doneTimestamping)
	go fillDatabase(ws.writerConnection, timedRecords, doneInsertingOnDatabase, doneTimestamping)

	return doneInsertingOnDatabase
}

// Gather non time stamped logs from various sources and timestamp them, making them ready
// for inserting on the database
func stampLogsWithTimeAndWaitUntilDatabaseIsFinished(timeConverter postfix.TimeConverter,
	records <-chan Record,
	timedRecords chan<- TimedRecord,
	done chan<- interface{}) {

	for r := range records {
		t := timeConverter.Convert(r.Header.Time)

		// do not bother the database thread if we have no payload to insert
		if r.Payload != nil {
			timedRecords <- TimedRecord{Time: t, Record: r}
		}
	}

	close(timedRecords)

	done <- true
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
	for _, handler := range payloadHandlers {
		if handler.counter(ws.readerConnection) > 0 {
			return true
		}
	}

	return false
}

func fillDatabase(db *sql.DB, c <-chan TimedRecord,
	doneInsertingOnDatabase chan<- interface{},
	doneTimestamping <-chan interface{}) {

	performInsertsIntoDbGroupingInTransactions(db, c, sqliteTransactionTime, func(tx *sql.Tx, r TimedRecord) error {
		inserter := findInserterForPayload(r.Record.Payload)

		if inserter != nil {
			return inserter(tx, r)
		}

		return nil
	})

	doneInsertingOnDatabase <- <-doneTimestamping
}

func performInsertsIntoDbGroupingInTransactions(db *sql.DB,
	c <-chan TimedRecord, timeout time.Duration,
	insertCb func(*sql.Tx, TimedRecord) error) {

	// This is the loop for the thread that inserts stuff in the logs database
	// It should be blocked only by writting to filesystem
	// It's executing during the entire program lifetime

	var tx *sql.Tx = nil
	var err error = nil

	countPerTransaction := 0

	startTransaction := func() {
		tx, err = db.Begin()

		if err != nil {
			log.Fatal(`Error preparing transaction:`, err)
		}

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

			err := insertCb(tx, r)

			if err != nil {
				log.Fatal("Error on database insertion:", err)
			}

			countPerTransaction += 1

			break

		case <-timeoutTimer:
			restartTransaction()
		}
	}
}

func buildInitialTime(db *sql.DB, defaultYear int, timezone *time.Location) (parser.Time, int) {
	// TODO: return max(defaultYear, year_read_from_database) as lightmeter might be restarted on the next year (rare, but possible)

	for _, handler := range payloadHandlers {
		if v, err := handler.lastTimeReader(db); err == nil {
			ts := time.Unix(v, 0).In(timezone)

			log.Println("Using initial time from database as:", ts)

			return parser.Time{
				Month:  ts.Month(),
				Day:    uint8(ts.Day()),
				Hour:   uint8(ts.Hour()),
				Minute: uint8(ts.Minute()),
				Second: uint8(ts.Second()),
			}, ts.Year()
		}
	}

	log.Println("Could not build initial time from database. Using default year", defaultYear)
	return parser.Time{}, defaultYear
}
