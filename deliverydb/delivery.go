// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0

package deliverydb

import (
	"database/sql"
	"errors"
	"github.com/rs/zerolog/log"
	_ "gitlab.com/lightmeter/controlcenter/deliverydb/migrations"
	"gitlab.com/lightmeter/controlcenter/domainmapping"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/migrator"
	parser "gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser"
	"gitlab.com/lightmeter/controlcenter/pkg/runner"
	"gitlab.com/lightmeter/controlcenter/tracking"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"path"
	"time"
)

type dbAction func(*sql.Tx) error

type DB struct {
	runner.CancelableRunner

	connPair  dbconn.ConnPair
	dbActions chan dbAction
}

const (
	filename = "logs.db"
)

func setupDomainMapping(conn dbconn.ConnPair, m *domainmapping.Mapper) error {
	// FIXME: this is an ugly workaround. Ideally the domain mapping should come from a virtual table,
	// computed from the domain mapped configuration.
	if _, err := conn.RwConn.Exec(`
	drop table if exists temp_domain_mapping;
	create table temp_domain_mapping(orig text, mapped text);
	create index temp_domain_mapping_index on temp_domain_mapping(orig, mapped);
	`); err != nil {
		return errorutil.Wrap(err)
	}

	f := func(orig, mapped string) error {
		if _, err := conn.RwConn.Exec(`insert into temp_domain_mapping(orig, mapped) values(?, ?)`, orig, mapped); err != nil {
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

	connPair, err := dbconn.NewConnPair(dbFilename)

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

	if err := setupDomainMapping(connPair, mapping); err != nil {
		return nil, errorutil.Wrap(err)
	}

	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	dbActions := make(chan dbAction, 1024*1000)

	db := DB{
		connPair:  connPair,
		dbActions: dbActions,
	}

	db.CancelableRunner = runner.NewCancelableRunner(func(done runner.DoneChan, cancel runner.CancelChan) {
		go func() {
			<-cancel
			close(dbActions)
		}()

		go func() {
			done <- func() error {
				if err := fillDatabase(connPair.RwConn, dbActions); err != nil {
					return errorutil.Wrap(err)
				}

				return nil
			}()
		}()
	})

	return &db, nil
}

func (db *DB) Close() error {
	if err := db.connPair.Close(); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

type resultsPublisher struct {
	dbActions chan<- dbAction
}

// TODO: cache all queries when application starts!

func getUniqueRemoteDomainNameId(tx *sql.Tx, domainName string) (int64, error) {
	var id int64
	err := tx.QueryRow(`select id from remote_domains where domain = ?`, domainName).Scan(&id)

	if err == nil {
		return id, nil
	}

	if !errors.Is(err, sql.ErrNoRows) {
		return 0, errorutil.Wrap(err)
	}

	// new domain. Should insert it
	result, err := tx.Exec(`insert into remote_domains(domain) values(?)`, domainName)
	if err != nil {
		return 0, errorutil.Wrap(err)
	}

	id, err = result.LastInsertId()
	if err != nil {
		return 0, errorutil.Wrap(err)
	}

	return id, nil
}

func getOptionalUniqueRemoteDomainNameId(tx *sql.Tx, domainName interface{}) (id int64, ok bool, err error) {
	if domainName == nil {
		return 0, false, nil
	}

	domainAsString := domainName.(string)

	if len(domainAsString) == 0 {
		return 0, false, nil
	}

	id, err = getUniqueRemoteDomainNameId(tx, domainAsString)
	if err != nil {
		return 0, false, errorutil.Wrap(err)
	}

	return id, true, nil
}

func getUniqueDeliveryServerID(tx *sql.Tx, hostname string) (int64, error) {
	// FIXME: this is almost copy&paste from getUniqueRemoteDomainNameId()!!!
	var id int64
	err := tx.QueryRow(`select id from delivery_server where hostname = ?`, hostname).Scan(&id)

	if err == nil {
		return id, nil
	}

	if !errors.Is(err, sql.ErrNoRows) {
		return 0, errorutil.Wrap(err)
	}

	// new domain. Should insert it
	result, err := tx.Exec(`insert into delivery_server(hostname) values(?)`, hostname)
	if err != nil {
		return 0, errorutil.Wrap(err)
	}

	id, err = result.LastInsertId()
	if err != nil {
		return 0, errorutil.Wrap(err)
	}

	return id, nil
}

func getUniqueMessageId(tx *sql.Tx, messageId string) (int64, error) {
	// FIXME: this is almost copy&paste from getUniqueRemoteDomainNameId()!!!
	var id int64
	err := tx.QueryRow(`select id from messageids where value = ?`, messageId).Scan(&id)

	if err == nil {
		return id, nil
	}

	if !errors.Is(err, sql.ErrNoRows) {
		return 0, errorutil.Wrap(err)
	}

	result, err := tx.Exec(`insert into messageids(value) values(?)`, messageId)
	if err != nil {
		return 0, errorutil.Wrap(err)
	}

	id, err = result.LastInsertId()
	if err != nil {
		return 0, errorutil.Wrap(err)
	}

	return id, nil
}

func getOptionalNextRelayId(tx *sql.Tx, relayName, relayIP interface{}, relayPort int64) (int64, bool, error) {
	// index order: name, ip, port
	if relayName == nil || relayIP == nil {
		return 0, false, nil
	}

	var id int64
	err := tx.QueryRow(`select id from next_relays where hostname = ? and ip = ? and port = ?`, relayName, relayIP, relayPort).Scan(&id)

	if err == nil {
		return id, true, nil
	}

	if !errors.Is(err, sql.ErrNoRows) {
		return 0, false, errorutil.Wrap(err)
	}

	result, err := tx.Exec(`insert into next_relays(hostname, ip, port) values(?, ?, ?)`, relayName.(string), relayIP, relayPort)
	if err != nil {
		return 0, false, errorutil.Wrap(err)
	}

	id, err = result.LastInsertId()
	if err != nil {
		return 0, false, errorutil.Wrap(err)
	}

	return id, true, nil
}

func insertMandatoryResultFields(tx *sql.Tx, tr tracking.Result) (sql.Result, error) {
	deliveryServerId, err := getUniqueDeliveryServerID(tx, tr[tracking.ResultDeliveryServerKey].(string))
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	senderDomainPartId, err := getUniqueRemoteDomainNameId(tx, tr[tracking.QueueSenderDomainPartKey].(string))
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	recipientDomainPartId, err := getUniqueRemoteDomainNameId(tx, tr[tracking.ResultRecipientDomainPartKey].(string))
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	messageId, err := getUniqueMessageId(tx, tr[tracking.QueueMessageIDKey].(string))
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	dir := func() tracking.MessageDirection {
		if asInt64, ok := tr[tracking.ResultMessageDirectionKey].(int64); ok {
			return tracking.MessageDirection(asInt64)
		}

		return tr[tracking.ResultMessageDirectionKey].(tracking.MessageDirection)
	}()

	status := func() parser.SmtpStatus {
		if asInt64, ok := tr[tracking.ResultStatusKey].(int64); ok {
			return parser.SmtpStatus(asInt64)
		}

		return tr[tracking.ResultStatusKey].(parser.SmtpStatus)
	}()

	result, err := tx.Exec(`
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
		status,
		tr[tracking.ResultDeliveryTimeKey].(int64),
		dir,
		senderDomainPartId,
		recipientDomainPartId,
		messageId,
		tr[tracking.ConnectionBeginKey],
		tr[tracking.QueueBeginKey],
		tr[tracking.QueueOriginalMessageSizeKey].(int64),
		tr[tracking.QueueProcessedMessageSizeKey].(int64),
		tr[tracking.QueueNRCPTKey],
		deliveryServerId,
		tr[tracking.ResultDelayKey],
		tr[tracking.ResultDelaySMTPDKey],
		tr[tracking.ResultDelayCleanupKey],
		tr[tracking.ResultDelayQmgrKey],
		tr[tracking.ResultDelaySMTPKey],
		tr[tracking.QueueSenderLocalPartKey].(string),
		tr[tracking.ResultRecipientLocalPartKey],
		tr[tracking.ConnectionClientHostnameKey],
		tr[tracking.ConnectionClientIPKey],
		tr[tracking.ResultDSNKey],
	)

	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	return result, nil
}

func buildAction(tr tracking.Result) func(tx *sql.Tx) error {
	return func(tx *sql.Tx) error {
		result, err := insertMandatoryResultFields(tx, tr)

		port := func() int64 {
			if tr[tracking.ResultRelayPortKey] == nil {
				return 0
			}

			return tr[tracking.ResultRelayPortKey].(int64)
		}()

		if err != nil {
			log.Warn().Msgf("%v", tr)
			return errorutil.Wrap(err)
		}

		rowId, err := result.LastInsertId()
		if err != nil {
			return errorutil.Wrap(err)
		}

		relayId, relayIdFound, err := getOptionalNextRelayId(tx, tr[tracking.ResultRelayNameKey], tr[tracking.ResultRelayIPKey], port)
		if err != nil {
			return errorutil.Wrap(err)
		}

		if relayIdFound {
			_, err = tx.Exec(`update deliveries set next_relay_id = ? where id = ?`, relayId, rowId)
			if err != nil {
				return errorutil.Wrap(err)
			}
		}

		origRecipientDomainPartId, origRecipientDomainPartFound, err := getOptionalUniqueRemoteDomainNameId(tx, tr[tracking.ResultOrigRecipientDomainPartKey])
		if err != nil {
			return errorutil.Wrap(err)
		}

		if origRecipientDomainPartFound {
			_, err = tx.Exec(`update deliveries set orig_recipient_domain_part_id = ? where id = ?`, origRecipientDomainPartId, rowId)
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
	var count int
	err := db.connPair.RoConn.QueryRow(`select count(*) from deliveries`).Scan(&count)
	errorutil.MustSucceed(err)

	return count > 0
}

func (db *DB) MostRecentLogTime() time.Time {
	var ts int64

	err := db.connPair.RoConn.QueryRow(`select delivery_ts from deliveries order by id desc limit 1`).Scan(&ts)

	if err != nil && errors.Is(err, sql.ErrNoRows) {
		return time.Time{}
	}

	errorutil.MustSucceed(err)

	return time.Unix(ts, 0).In(time.UTC)
}

func (db *DB) ReadConnection() dbconn.RoConn {
	return db.connPair.RoConn
}

func fillDatabase(conn dbconn.RwConn, dbActions <-chan dbAction) error {
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

		if err := action(tx); err != nil {
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
