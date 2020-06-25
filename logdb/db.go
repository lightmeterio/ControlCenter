package logdb

import (
	"database/sql"
	"log"
	"path"
	"time"

	"gitlab.com/lightmeter/controlcenter/data"
	"gitlab.com/lightmeter/controlcenter/data/postfix"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/util"
	parser "gitlab.com/lightmeter/postfix-log-parser"
)

var (
	TransactionTime time.Duration = 1000 * time.Millisecond
)

const (
	filename         = "logs.db"
	recordsQueueSize = 100
)

type ChannelBasedPublisher struct {
	Channel chan<- data.Record
}

func (pub *ChannelBasedPublisher) Publish(status data.Record) {
	pub.Channel <- status
}

func (pub *ChannelBasedPublisher) Close() {
	close(pub.Channel)
}

type DB struct {
	config   data.Config
	connPair dbconn.ConnPair

	writerConnection *sql.DB

	// connection that is unable to do any changes to the database (like inserts, updates, etc.)
	readerConnection *sql.DB

	records chan data.Record
}

func setupWriterConn(conn *sql.DB) error {
	if err := createTables(conn); err != nil {
		log.Println("Error creating table or indexes:", err)
		return util.WrapError(err)
	}

	return nil
}

func createTables(db *sql.DB) error {
	for _, handler := range payloadHandlers {
		if err := handler.creator(db); err != nil {
			return util.WrapError(err)
		}
	}

	return nil
}

func Open(workspaceDirectory string, config data.Config) (DB, error) {
	dbFilename := path.Join(workspaceDirectory, filename)

	connPair, err := dbconn.NewConnPair(dbFilename)

	if err != nil {
		return DB{}, util.WrapError(err)
	}

	defer func() {
		if err != nil {
			util.MustSucceed(connPair.Close(), "Closing connection on error")
		}
	}()

	err = setupWriterConn(connPair.RwConn)

	if err != nil {
		return DB{}, util.WrapError(err)
	}

	return DB{
		config:   config,
		connPair: connPair,
		records:  make(chan data.Record, recordsQueueSize),
	}, nil
}

func (db *DB) ReadConnection() *sql.DB {
	return db.connPair.RoConn
}

func (db *DB) NewPublisher() data.Publisher {
	return &ChannelBasedPublisher{db.records}
}

// Obtain the most recent time inserted in the database,
// or a zero'd time in case case no value has been found
func (db *DB) MostRecentLogTime() time.Time {
	return buildInitialTime(db.connPair.RoConn, db.config.DefaultYear, db.config.Location)
}

func (db *DB) Run() <-chan interface{} {
	doneTimestamping := make(chan interface{})
	doneInsertingOnDatabase := make(chan interface{})

	newYearNotifier := func(year int, old parser.Time, new parser.Time) {
		log.Println("Bumping year", year, ", old:", old, ", new:", new)
	}

	time, year := func(ts time.Time) (parser.Time, int) {
		if ts.IsZero() {
			return parser.Time{}, db.config.DefaultYear
		}

		log.Println("Using initial time from database as:", ts)

		return parser.Time{
			Month:  ts.Month(),
			Day:    uint8(ts.Day()),
			Hour:   uint8(ts.Hour()),
			Minute: uint8(ts.Minute()),
			Second: uint8(ts.Second()),
		}, ts.Year()
	}(db.MostRecentLogTime())

	converter := postfix.NewTimeConverter(time, year, db.config.Location, newYearNotifier)
	timedRecords := make(chan data.TimedRecord, recordsQueueSize)

	go stampLogsWithTimeAndWaitUntilDatabaseIsFinished(converter, db.records, timedRecords, doneTimestamping)
	go fillDatabase(db.connPair.RwConn, timedRecords, doneInsertingOnDatabase, doneTimestamping)

	return doneInsertingOnDatabase
}

// Gather non time stamped logs from various sources and timestamp them, making them ready
// for inserting on the database
func stampLogsWithTimeAndWaitUntilDatabaseIsFinished(timeConverter postfix.TimeConverter,
	records <-chan data.Record,
	timedRecords chan<- data.TimedRecord,
	done chan<- interface{}) {

	for r := range records {
		t := timeConverter.Convert(r.Header.Time)

		// do not bother the database thread if we have no payload to insert
		if r.Payload != nil {
			timedRecords <- data.TimedRecord{Time: t, Record: r}
		}
	}

	close(timedRecords)

	done <- true
}

func (db *DB) Close() error {
	return db.connPair.Close()
}

func (db *DB) HasLogs() bool {
	for _, handler := range payloadHandlers {
		if handler.counter(db.connPair.RoConn) > 0 {
			return true
		}
	}

	return false
}

func fillDatabase(db *sql.DB, c <-chan data.TimedRecord,
	doneInsertingOnDatabase chan<- interface{},
	doneTimestamping <-chan interface{}) {

	performInsertsIntoDbGroupingInTransactions(db, c, TransactionTime, func(tx *sql.Tx, r data.TimedRecord) error {
		inserter := findInserterForPayload(r.Record.Payload)

		if inserter != nil {
			return inserter(tx, r)
		}

		return nil
	})

	doneInsertingOnDatabase <- <-doneTimestamping
}

func performInsertsIntoDbGroupingInTransactions(db *sql.DB,
	c <-chan data.TimedRecord, timeout time.Duration,
	insertCb func(*sql.Tx, data.TimedRecord) error) {

	// This is the loop for the thread that inserts stuff in the logs database
	// It should be blocked only by writing to filesystem
	// It's executing during the entire program lifetime

	var tx *sql.Tx = nil
	var err error = nil

	countPerTransaction := 0

	startTransaction := func() {
		tx, err = db.Begin()
		util.MustSucceed(err, "Preparing transaction")
	}

	closeTransaction := func() {
		if countPerTransaction == 0 {
			util.MustSucceed(tx.Rollback(), "Rolling back empty transaction")
			return
		}

		// NOTE: improve it to be used for benchmarking
		log.Println("Inserted", countPerTransaction, "rows in a transaction")

		countPerTransaction = 0

		util.MustSucceed(tx.Commit(), "Committing transaction")
	}

	restartTransaction := func() {
		closeTransaction()
		startTransaction()
	}

	ticker := time.NewTicker(timeout)

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

			util.MustSucceed(insertCb(tx, r), "Database insertion")
			countPerTransaction += 1
			break
		case <-ticker.C:
			restartTransaction()
		}
	}
}

func buildInitialTime(db *sql.DB, defaultYear int, timezone *time.Location) time.Time {
	// TODO: return max(defaultYear, year_read_from_database) as lightmeter might be restarted on the next year (rare, but possible)

	for _, handler := range payloadHandlers {
		if v, err := handler.lastTimeReader(db); err == nil {
			ts := time.Unix(v, 0).In(timezone)
			return ts
		}
	}

	log.Println("Could not build initial time from database. Using default year", defaultYear)
	return time.Time{}
}
