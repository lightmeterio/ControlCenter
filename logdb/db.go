package logdb

import (
	"database/sql"
	"errors"
	"log"
	"path"
	"time"

	"gitlab.com/lightmeter/controlcenter/data"
	"gitlab.com/lightmeter/controlcenter/data/postfix"
	_ "gitlab.com/lightmeter/controlcenter/lmsqlite3"
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
	config data.Config

	writerConnection *sql.DB

	// connection that is unable to do any changes to the database (like inserts, updates, etc.)
	readerConnection *sql.DB

	records chan data.Record
}

func attachDatabase(conn *sql.DB, filename, schemaName string, flags string) error {
	q := `attach database 'file:` + filename + `?` + flags + `' as ` + schemaName

	_, err := conn.Exec(q)

	if err != nil {
		return err
	}

	return nil
}

func detachDB(db *sql.DB, schema string) error {
	_, err := db.Exec(`detach database ` + schema)
	return err
}

func createWriter(dbFilename string, config data.Config) (*sql.DB, error) {
	conn, err := sql.Open("lm_sqlite3", `file:`+dbFilename+`?mode=rwc&cache=private&_loc=auto&_journal=WAL`)

	if err != nil {
		return nil, err
	}

	defer func() {
		if err != nil {
			util.MustSucceed(conn.Close(), "Closing RW connection on error")
		}
	}()

	err = createTables(conn)

	if err != nil {
		log.Println("Error creating table or indexes:", err)
		return nil, err
	}

	return conn, nil
}

func createReader(dbFilename string, config data.Config) (*sql.DB, error) {
	conn, err := sql.Open("lm_sqlite3", `file:`+dbFilename+`?mode=ro&cache=shared&_query_only=true&_loc=auto&_journal=WAL`)

	if err != nil {
		return nil, err
	}

	defer func() {
		if err != nil {
			util.MustSucceed(conn.Close(), "Closing RO Connection on error")
		}
	}()

	return conn, nil
}

func createTables(db *sql.DB) error {
	for _, handler := range payloadHandlers {
		if err := handler.creator(db); err != nil {
			return err
		}
	}

	return nil
}

func Open(workspaceDirectory string, config data.Config) (DB, error) {
	dbFilename := path.Join(workspaceDirectory, filename)

	writerConnection, err := createWriter(dbFilename, config)

	if err != nil {
		return DB{}, err
	}

	defer func() {
		if err != nil {
			util.MustSucceed(writerConnection.Close(), "Closing writer on error")
		}
	}()

	readerConnection, err := createReader(dbFilename, config)

	if err != nil {
		return DB{}, err
	}

	defer func() {
		if err != nil {
			util.MustSucceed(readerConnection.Close(), "Closing reader on error")
		}
	}()

	return DB{
		config:           config,
		writerConnection: writerConnection,
		readerConnection: readerConnection,
		records:          make(chan data.Record, recordsQueueSize),
	}, nil
}

func (db *DB) ReadConnection() *sql.DB {
	return db.readerConnection
}

func (db *DB) NewPublisher() data.Publisher {
	return &ChannelBasedPublisher{db.records}
}

// Obtain the most recent time inserted in the database,
// or a zero'd time in case case no value has been found
func (db *DB) MostRecentLogTime() time.Time {
	return buildInitialTime(db.readerConnection, db.config.DefaultYear, db.config.Location)
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
	go fillDatabase(db.writerConnection, timedRecords, doneInsertingOnDatabase, doneTimestamping)

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
	errReader := db.readerConnection.Close()
	errWriter := db.writerConnection.Close()

	if errWriter != nil || errReader != nil {
		return errors.New("error closing database: writer:(" + errWriter.Error() + "), reader: (" + errReader.Error() + ")")
	}

	return nil
}

func (db *DB) HasLogs() bool {
	for _, handler := range payloadHandlers {
		if handler.counter(db.readerConnection) > 0 {
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
