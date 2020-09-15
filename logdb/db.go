package logdb

import (
	"database/sql"
	"log"
	"path"
	"time"

	"gitlab.com/lightmeter/controlcenter/data"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/util"
)

var (
	TransactionTime time.Duration = 1000 * time.Millisecond
)

const (
	filename         = "logs.db"
	recordsQueueSize = 4096
)

type ChannelBasedPublisher struct {
	Channel chan<- data.Record
}

func (pub *ChannelBasedPublisher) Publish(status data.Record) {
	if status.Payload != nil {
		pub.Channel <- status
	}
}

func (pub *ChannelBasedPublisher) Close() {
	close(pub.Channel)
}

type Config struct {
	Location *time.Location
}

type DB struct {
	config   Config
	connPair dbconn.ConnPair
	records  chan data.Record
}

func setupWriterConn(conn dbconn.RwConn) error {
	if err := createTables(conn); err != nil {
		log.Println("Error creating table or indexes:", err)
		return util.WrapError(err)
	}

	return nil
}

func createTables(db dbconn.RwConn) error {
	for _, handler := range payloadHandlers {
		if err := handler.creator(db); err != nil {
			return util.WrapError(err)
		}
	}
	return nil
}

func Open(workspaceDirectory string, config Config) (DB, error) {
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

func (db *DB) ReadConnection() dbconn.RoConn {
	return db.connPair.RoConn
}

func (db *DB) NewPublisher() data.Publisher {
	return &ChannelBasedPublisher{db.records}
}

// Obtain the most recent time inserted in the database,
// or a zero'd time in case case no value has been found
func (db *DB) MostRecentLogTime() time.Time {
	return buildInitialTime(db.connPair.RoConn, db.config.Location)
}

func (db *DB) Run() <-chan interface{} {
	doneInsertingOnDatabase := make(chan interface{})
	go fillDatabase(db.connPair.RwConn, db.records, doneInsertingOnDatabase)
	return doneInsertingOnDatabase
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

func fillDatabase(db dbconn.RwConn, c <-chan data.Record,
	doneInsertingOnDatabase chan<- interface{}) {

	lastTime := time.Time{}

	performInsertsIntoDbGroupingInTransactions(db, c, TransactionTime, func(tx *sql.Tx, r data.Record) error {
		if lastTime.After(r.Time) {
			log.Panicln("Out of order log insertion in the database. Old:", lastTime, ", new:", r.Time)
		}

		lastTime = r.Time

		inserter := findInserterForPayload(r.Payload)

		if inserter != nil {
			return inserter(tx, r)
		}

		return nil
	})

	doneInsertingOnDatabase <- nil
}

func performInsertsIntoDbGroupingInTransactions(db dbconn.RwConn,
	c <-chan data.Record, timeout time.Duration,
	insertCb func(*sql.Tx, data.Record) error) {

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

func buildInitialTime(db dbconn.RoConn, timezone *time.Location) time.Time {
	r := int64(0)

	for _, handler := range payloadHandlers {
		if v, err := handler.lastTimeReader(db); err == nil && v > r {
			r = v
		}
	}

	if r == 0 {
		return time.Time{}
	}

	return time.Unix(r, 0).In(timezone)
}
