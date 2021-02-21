// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package deliverydb

import (
	"database/sql"
	"errors"
	"github.com/rs/zerolog/log"
	_ "gitlab.com/lightmeter/controlcenter/deliverydb/migrations"
	"gitlab.com/lightmeter/controlcenter/domainmapping"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/migrator"
	"gitlab.com/lightmeter/controlcenter/pkg/runner"
	"gitlab.com/lightmeter/controlcenter/tracking"
	"gitlab.com/lightmeter/controlcenter/util/closeutil"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"path"
	"time"
)

type dbAction func(*sql.Tx, preparedStmts) error

type DB struct {
	runner.CancelableRunner
	closeutil.Closers

	connPair  *dbconn.PooledPair
	dbActions chan dbAction
	stmts     preparedStmts
}

const (
	filename = "logs.db"
)

type stmtKey uint

const (
	selectIdFromRemoteDomain stmtKey = iota
	insertRemoteDomain
	selectDeliveryServerByHostname
	insertDeliveryServer
	selectMessageIdsByValue
	insertMessageId
	selectNextRelays
	insertNextRelay
	insertDelivery
	updateDeliveryWithRelay
	updateDeliveryWithOrigRecipient

	lastStmtKey
)

type preparedStmts [lastStmtKey]*sql.Stmt

var stmtsText = map[stmtKey]string{
	selectIdFromRemoteDomain:       `select id from remote_domains where domain = ?`,
	insertRemoteDomain:             `insert into remote_domains(domain) values(?)`,
	selectDeliveryServerByHostname: `select id from delivery_server where hostname = ?`,
	insertDeliveryServer:           `insert into delivery_server(hostname) values(?)`,
	selectMessageIdsByValue:        `select id from messageids where value = ?`,
	insertMessageId:                `insert into messageids(value) values(?)`,
	selectNextRelays:               `select id from next_relays where hostname = ? and ip = ? and port = ?`,
	insertNextRelay:                `insert into next_relays(hostname, ip, port) values(?, ?, ?)`,
	insertDelivery: `
insert into deliveries(
	status,
	delivery_ts,
	direction,
	sender_domain_part_id,
	recipient_domain_part_id,
	message_id,
	conn_ts_begin,
	queue_ts_begin,
	orig_msg_size,
	processed_msg_size,
	nrcpt,
	delivery_server_id,
	delay,
	delay_smtpd,
	delay_cleanup,
	delay_qmgr,
	delay_smtp,
	sender_local_part,
	recipient_local_part,
	client_hostname,
	client_ip,
	dsn)
values(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
	updateDeliveryWithRelay:         `update deliveries set next_relay_id = ? where id = ?`,
	updateDeliveryWithOrigRecipient: `update deliveries set orig_recipient_domain_part_id = ? where id = ?`,
}

// TODO: close such statements when the tracker is deleted!!!
func prepareRwStmts(conn dbconn.RwConn) (preparedStmts, error) {
	stmts := preparedStmts{}

	for k, v := range stmtsText {
		//nolint:sqlclosecheck
		stmt, err := conn.Prepare(v)
		if err != nil {
			return preparedStmts{}, errorutil.Wrap(err)
		}

		stmts[k] = stmt
	}

	return stmts, nil
}

func setupDomainMapping(conn dbconn.RwConn, m *domainmapping.Mapper) error {
	// FIXME: this is an ugly workaround. Ideally the domain mapping should come from a virtual table,
	// computed from the domain mapped configuration.
	if _, err := conn.Exec(`
	drop table if exists temp_domain_mapping;
	create table temp_domain_mapping(orig text, mapped text);
	create index temp_domain_mapping_index on temp_domain_mapping(orig, mapped);
	`); err != nil {
		return errorutil.Wrap(err)
	}

	f := func(orig, mapped string) error {
		if _, err := conn.Exec(`insert into temp_domain_mapping(orig, mapped) values(?, ?)`, orig, mapped); err != nil {
			return errorutil.Wrap(err)
		}

		return nil
	}

	if err := m.ForEach(f); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func New(workspace string, mapping *domainmapping.Mapper) (*DB, error) {
	dbFilename := path.Join(workspace, filename)

	connPair, err := dbconn.Open(dbFilename, 10)

	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	defer func() {
		if err != nil {
			errorutil.MustSucceed(connPair.Close(), "Closing connection on error")
		}
	}()

	if err := migrator.Run(connPair.RwConn.DB, "deliverydb"); err != nil {
		return nil, errorutil.Wrap(err)
	}

	if err := setupDomainMapping(connPair.RwConn, mapping); err != nil {
		return nil, errorutil.Wrap(err)
	}

	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	stmts, err := prepareRwStmts(connPair.RwConn)
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	dbActions := make(chan dbAction, 1024*1000)

	db := DB{
		connPair:  connPair,
		dbActions: dbActions,
		Closers:   closeutil.New(connPair),
		stmts:     stmts,
	}

	db.CancelableRunner = runner.NewCancelableRunner(func(done runner.DoneChan, cancel runner.CancelChan) {
		go func() {
			<-cancel
			close(dbActions)
		}()

		go func() {
			done <- func() error {
				if err := fillDatabase(connPair.RwConn, stmts, dbActions); err != nil {
					return errorutil.Wrap(err)
				}

				return nil
			}()
		}()
	})

	return &db, nil
}

type resultsPublisher struct {
	dbActions chan<- dbAction
}

func getUniquePropertyFromAnotherTable(tx *sql.Tx, selectStmt, insertStmt *sql.Stmt, args ...interface{}) (int64, error) {
	var id int64

	stmt := tx.Stmt(selectStmt)

	defer func() {
		errorutil.MustSucceed(stmt.Close())
	}()

	err := stmt.QueryRow(args...).Scan(&id)

	if err == nil {
		return id, nil
	}

	if !errors.Is(err, sql.ErrNoRows) {
		return 0, errorutil.Wrap(err)
	}

	stmt = tx.Stmt(insertStmt)

	defer func() {
		errorutil.MustSucceed(stmt.Close())
	}()

	result, err := stmt.Exec(args...)
	if err != nil {
		return 0, errorutil.Wrap(err)
	}

	id, err = result.LastInsertId()
	if err != nil {
		return 0, errorutil.Wrap(err)
	}

	return id, nil
}

func getUniqueRemoteDomainNameId(tx *sql.Tx, stmts preparedStmts, domainName string) (int64, error) {
	id, err := getUniquePropertyFromAnotherTable(tx, stmts[selectIdFromRemoteDomain], stmts[insertRemoteDomain], domainName)

	if err != nil {
		return 0, errorutil.Wrap(err)
	}

	return id, nil
}

func getOptionalUniqueRemoteDomainNameId(tx *sql.Tx, stmts preparedStmts, domainName tracking.ResultEntry) (id int64, ok bool, err error) {
	if domainName.IsNone() {
		return 0, false, nil
	}

	domainAsString := domainName.Text()

	if len(domainAsString) == 0 {
		return 0, false, nil
	}

	id, err = getUniqueRemoteDomainNameId(tx, stmts, domainAsString)
	if err != nil {
		return 0, false, errorutil.Wrap(err)
	}

	return id, true, nil
}

func getUniqueDeliveryServerID(tx *sql.Tx, stmts preparedStmts, hostname string) (int64, error) {
	id, err := getUniquePropertyFromAnotherTable(tx, stmts[selectDeliveryServerByHostname], stmts[insertDeliveryServer], hostname)
	if err != nil {
		return 0, errorutil.Wrap(err)
	}

	return id, nil
}

func getUniqueMessageId(tx *sql.Tx, stmts preparedStmts, messageId string) (int64, error) {
	id, err := getUniquePropertyFromAnotherTable(tx, stmts[selectMessageIdsByValue], stmts[insertMessageId], messageId)

	if err != nil {
		return 0, errorutil.Wrap(err)
	}

	return id, nil
}

func getOptionalNextRelayId(tx *sql.Tx, stmts preparedStmts, relayName, relayIP tracking.ResultEntry, relayPort int64) (int64, bool, error) {
	// index order: name, ip, port
	if relayName.IsNone() || relayIP.IsNone() {
		return 0, false, nil
	}

	id, err := getUniquePropertyFromAnotherTable(tx, stmts[selectNextRelays], stmts[insertNextRelay], relayName.Text(), relayIP.Blob(), relayPort)
	if err != nil {
		return 0, false, errorutil.Wrap(err)
	}

	return id, true, nil
}

func insertMandatoryResultFields(tx *sql.Tx, stmts preparedStmts, tr tracking.Result) (sql.Result, error) {
	deliveryServerId, err := getUniqueDeliveryServerID(tx, stmts, tr[tracking.ResultDeliveryServerKey].Text())
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	senderDomainPartId, err := getUniqueRemoteDomainNameId(tx, stmts, tr[tracking.QueueSenderDomainPartKey].Text())
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	recipientDomainPartId, err := getUniqueRemoteDomainNameId(tx, stmts, tr[tracking.ResultRecipientDomainPartKey].Text())
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	messageId, err := getUniqueMessageId(tx, stmts, tr[tracking.QueueMessageIDKey].Text())
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	dir := tr[tracking.ResultMessageDirectionKey].Int64()

	status := tr[tracking.ResultStatusKey].Int64()

	stmt := tx.Stmt(stmts[insertDelivery])

	defer func() {
		errorutil.MustSucceed(stmt.Close())
	}()

	result, err := stmt.Exec(
		status,
		tr[tracking.ResultDeliveryTimeKey].Int64(),
		dir,
		senderDomainPartId,
		recipientDomainPartId,
		messageId,
		tr[tracking.ConnectionBeginKey].Int64(),
		tr[tracking.QueueBeginKey].Int64(),
		tr[tracking.QueueOriginalMessageSizeKey].Int64(),
		tr[tracking.QueueProcessedMessageSizeKey].Int64(),
		tr[tracking.QueueNRCPTKey].Int64(),
		deliveryServerId,
		tr[tracking.ResultDelayKey].Float64(),
		tr[tracking.ResultDelaySMTPDKey].Float64(),
		tr[tracking.ResultDelayCleanupKey].Float64(),
		tr[tracking.ResultDelayQmgrKey].Float64(),
		tr[tracking.ResultDelaySMTPKey].Float64(),
		tr[tracking.QueueSenderLocalPartKey].Text(),
		tr[tracking.ResultRecipientLocalPartKey].Text(),
		tr[tracking.ConnectionClientHostnameKey].Text(),
		tr[tracking.ConnectionClientIPKey].Blob(),
		tr[tracking.ResultDSNKey].Text(),
	)

	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	return result, nil
}

func buildAction(tr tracking.Result) func(*sql.Tx, preparedStmts) error {
	return func(tx *sql.Tx, stmts preparedStmts) error {
		result, err := insertMandatoryResultFields(tx, stmts, tr)

		port := func() int64 {
			if tr[tracking.ResultRelayPortKey].IsNone() {
				return 0
			}

			return tr[tracking.ResultRelayPortKey].Int64()
		}()

		if err != nil {
			log.Warn().Msgf("%v", tr)
			return errorutil.Wrap(err)
		}

		rowId, err := result.LastInsertId()
		if err != nil {
			return errorutil.Wrap(err)
		}

		relayId, relayIdFound, err := getOptionalNextRelayId(tx, stmts, tr[tracking.ResultRelayNameKey], tr[tracking.ResultRelayIPKey], port)
		if err != nil {
			return errorutil.Wrap(err)
		}

		if relayIdFound {
			stmt := tx.Stmt(stmts[updateDeliveryWithRelay])

			defer func() {
				errorutil.MustSucceed(stmt.Close())
			}()

			_, err = stmt.Exec(relayId, rowId)
			if err != nil {
				return errorutil.Wrap(err)
			}
		}

		origRecipientDomainPartId, origRecipientDomainPartFound, err := getOptionalUniqueRemoteDomainNameId(tx, stmts, tr[tracking.ResultOrigRecipientDomainPartKey])
		if err != nil {
			return errorutil.Wrap(err)
		}

		if origRecipientDomainPartFound {
			stmt := tx.Stmt(stmts[updateDeliveryWithOrigRecipient])

			defer func() {
				errorutil.MustSucceed(stmt.Close())
			}()

			_, err = stmt.Exec(origRecipientDomainPartId, rowId)
			if err != nil {
				return errorutil.Wrap(err)
			}
		}

		return nil
	}
}

func (p *resultsPublisher) Publish(r tracking.Result) {
	p.dbActions <- buildAction(r)
}

func (db *DB) ResultsPublisher() tracking.ResultPublisher {
	return &resultsPublisher{dbActions: db.dbActions}
}

func (db *DB) HasLogs() bool {
	conn, release := db.connPair.RoConnPool.Acquire()

	defer release()

	var count int
	err := conn.QueryRow(`select count(*) from deliveries`).Scan(&count)
	errorutil.MustSucceed(err)

	return count > 0
}

func (db *DB) MostRecentLogTime() time.Time {
	conn, release := db.connPair.RoConnPool.Acquire()

	defer release()

	var ts int64

	err := conn.QueryRow(`select delivery_ts from deliveries order by id desc limit 1`).Scan(&ts)

	if err != nil && errors.Is(err, sql.ErrNoRows) {
		return time.Time{}
	}

	errorutil.MustSucceed(err)

	return time.Unix(ts, 0).In(time.UTC)
}

func (db *DB) ConnPool() *dbconn.RoPool {
	return db.connPair.RoConnPool
}

func fillDatabase(conn dbconn.RwConn, stmts preparedStmts, dbActions <-chan dbAction) error {
	var (
		tx                  *sql.Tx = nil
		countPerTransaction int64
	)

	startTransaction := func() error {
		var err error
		if tx, err = conn.Begin(); err != nil {
			return errorutil.Wrap(err)
		}

		return nil
	}

	closeTransaction := func() error {
		// no transaction to commit
		if tx == nil {
			return nil
		}

		// NOTE: improve it to be used for benchmarking
		log.Info().Msgf("Inserted %d rows in a transaction", countPerTransaction)

		if err := tx.Commit(); err != nil {
			return errorutil.Wrap(err)
		}

		countPerTransaction = 0
		tx = nil

		return nil
	}

	tryToDoAction := func(action dbAction) error {
		if tx == nil {
			if err := startTransaction(); err != nil {
				return errorutil.Wrap(err)
			}
		}

		if err := action(tx, stmts); err != nil {
			return errorutil.Wrap(err)
		}

		countPerTransaction++

		return nil
	}

	ticker := time.NewTicker(500 * time.Millisecond)

	for {
		select {
		case <-ticker.C:
			if err := closeTransaction(); err != nil {
				return errorutil.Wrap(err)
			}
		case action, ok := <-dbActions:
			{
				if !ok {
					log.Info().Msg("Committing because there's nothing left")

					// cancel() has been called!!!
					if err := closeTransaction(); err != nil {
						return errorutil.Wrap(err)
					}

					return nil
				}

				if err := tryToDoAction(action); err != nil {
					return errorutil.Wrap(err)
				}
			}
		}
	}
}
