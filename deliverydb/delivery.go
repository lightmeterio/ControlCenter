// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package deliverydb

import (
	"database/sql"
	"errors"
	_ "gitlab.com/lightmeter/controlcenter/deliverydb/migrations"
	"gitlab.com/lightmeter/controlcenter/domainmapping"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/pkg/closers"
	"gitlab.com/lightmeter/controlcenter/pkg/dbrunner"
	parser "gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser"
	"gitlab.com/lightmeter/controlcenter/tracking"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"time"
)

type dbAction = dbrunner.Action

type DB struct {
	*dbrunner.Runner
	closers.Closers

	connPair *dbconn.PooledPair
}

type stmtKey = int

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
	insertQueue
	insertQueueDeliveryAttempt
	findQueueByName
	insertQueueParenting
	insertExpiredQueue

	selectOldDeliveries
	deleteOldDeliveries
	selectQueueIdForDeliveryId
	countDeliveriesWithQueue
	deleteDeliveryQueueById
	deleteExpiredQueuesByQueueId
	deleteQueueParentingByQueueId
	deleteQueueById
	countDeliveriesWithMessageId
	deleteMessageIdById

	lastStmtKey
)

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
	insertQueue:                     `insert into queues(name) values(?)`,
	insertQueueDeliveryAttempt:      `insert into delivery_queue(queue_id, delivery_id) values(?, ?)`,
	findQueueByName:                 `select id from queues where name = ?`,
	insertQueueParenting:            `insert into queue_parenting(parent_queue_id, child_queue_id, type) values(?, ?, ?)`,
	insertExpiredQueue:              `insert into expired_queues(queue_id, expired_ts) values(?, ?)`,
	selectOldDeliveries: `with time_cut as (
		select
			(delivery_ts - ?) as v
		from
			deliveries
		order by
			delivery_ts desc limit 1
	)
	select
		deliveries.id, deliveries.message_id, deliveries.delivery_ts
	from
		deliveries join time_cut
			on deliveries.delivery_ts < time_cut.v
	limit ?`,
	deleteOldDeliveries: `delete from deliveries where id = ?`,
	selectQueueIdForDeliveryId: `select
	delivery_queue.id, delivery_queue.queue_id
from
	delivery_queue
where
	delivery_queue.delivery_id = ?	`,
	countDeliveriesWithQueue: `select
	count(*)
from
	deliveries join delivery_queue on delivery_queue.delivery_id = deliveries.id
where
	delivery_queue.queue_id = ?`,
	deleteDeliveryQueueById:       `delete from delivery_queue where id = ?`,
	deleteExpiredQueuesByQueueId:  `delete from expired_queues where queue_id = ?`,
	deleteQueueParentingByQueueId: `delete from queue_parenting where parent_queue_id = ? or child_queue_id = ?`,
	deleteQueueById:               `delete from queues where id = ?`,
	countDeliveriesWithMessageId:  `select count(*) from deliveries where message_id = ? and delivery_ts < ?`,
	deleteMessageIdById:           `delete from messageids where id = ?`,
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

func New(connPair *dbconn.PooledPair, mapping *domainmapping.Mapper) (*DB, error) {
	if err := setupDomainMapping(connPair.RwConn, mapping); err != nil {
		return nil, errorutil.Wrap(err)
	}

	stmts := dbconn.BuildPreparedStmts(lastStmtKey)

	if err := dbconn.PrepareRwStmts(stmtsText, connPair.RwConn, &stmts); err != nil {
		return nil, errorutil.Wrap(err)
	}

	const (
		// ~3 months. TODO: make it configurable
		maxAge            = (time.Hour * 24 * 30 * 3)
		cleaningBatchSize = 10000
		cleaningFrequency = time.Second * 30
	)

	return &DB{
		connPair: connPair,
		Runner:   dbrunner.New(500*time.Millisecond, 1024*1000, connPair.RwConn, stmts, cleaningFrequency, makeCleanAction(maxAge, cleaningBatchSize)),
		Closers:  closers.New(stmts),
	}, nil
}

type resultsPublisher struct {
	dbActions chan<- dbAction
}

func getUniquePropertyFromAnotherTable(tx *sql.Tx, selectStmt, insertStmt *sql.Stmt, args ...interface{}) (int64, error) {
	var id int64

	err := selectStmt.QueryRow(args...).Scan(&id)

	if err == nil {
		return id, nil
	}

	if !errors.Is(err, sql.ErrNoRows) {
		return 0, errorutil.Wrap(err)
	}

	result, err := insertStmt.Exec(args...)
	if err != nil {
		return 0, errorutil.Wrap(err)
	}

	id, err = result.LastInsertId()
	if err != nil {
		return 0, errorutil.Wrap(err)
	}

	return id, nil
}

func getUniqueRemoteDomainNameId(tx *sql.Tx, stmts dbconn.TxPreparedStmts, domainName string) (int64, error) {
	//nolint:sqlclosecheck
	id, err := getUniquePropertyFromAnotherTable(tx, stmts.Get(selectIdFromRemoteDomain), stmts.Get(insertRemoteDomain), domainName)

	if err != nil {
		return 0, errorutil.Wrap(err)
	}

	return id, nil
}

func getOptionalUniqueRemoteDomainNameId(tx *sql.Tx, stmts dbconn.TxPreparedStmts, domainName tracking.ResultEntry) (id int64, ok bool, err error) {
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

func getUniqueDeliveryServerID(tx *sql.Tx, stmts dbconn.TxPreparedStmts, hostname string) (int64, error) {
	//nolint:sqlclosecheck
	id, err := getUniquePropertyFromAnotherTable(tx, stmts.Get(selectDeliveryServerByHostname), stmts.Get(insertDeliveryServer), hostname)
	if err != nil {
		return 0, errorutil.Wrap(err)
	}

	return id, nil
}

func getUniqueMessageId(tx *sql.Tx, stmts dbconn.TxPreparedStmts, messageId string) (int64, error) {
	//nolint:sqlclosecheck
	id, err := getUniquePropertyFromAnotherTable(tx, stmts.Get(selectMessageIdsByValue), stmts.Get(insertMessageId), messageId)

	if err != nil {
		return 0, errorutil.Wrap(err)
	}

	return id, nil
}

func getOptionalNextRelayId(tx *sql.Tx, stmts dbconn.TxPreparedStmts, relayName, relayIP tracking.ResultEntry, relayPort int64) (int64, bool, error) {
	// index order: name, ip, port
	if relayName.IsNone() || relayIP.IsNone() {
		return 0, false, nil
	}

	//nolint:sqlclosecheck
	id, err := getUniquePropertyFromAnotherTable(tx, stmts.Get(selectNextRelays), stmts.Get(insertNextRelay), relayName.Text(), relayIP.Blob(), relayPort)
	if err != nil {
		return 0, false, errorutil.Wrap(err)
	}

	return id, true, nil
}

func insertMandatoryResultFields(tx *sql.Tx, stmts dbconn.TxPreparedStmts, tr tracking.Result) (sql.Result, error) {
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

	//nolint:sqlclosecheck
	result, err := stmts.Get(insertDelivery).Exec(
		status,
		tr[tracking.ResultDeliveryTimeKey].Int64(),
		dir,
		senderDomainPartId,
		recipientDomainPartId,
		messageId,
		valueOrNil(tr[tracking.ConnectionBeginKey]),
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
		valueOrNil(tr[tracking.ConnectionClientHostnameKey]),
		valueOrNil(tr[tracking.ConnectionClientIPKey]),
		tr[tracking.ResultDSNKey].Text(),
	)

	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	return result, nil
}

// FIXME: this is a workaround due an issue in the parser on obtaining the connection
// information on NOQUEUE, afaik
func valueOrNil(e tracking.ResultEntry) interface{} {
	return e.ValueOrNil()
}

func buildAction(tr tracking.Result) func(*sql.Tx, dbconn.TxPreparedStmts) error {
	return func(tx *sql.Tx, stmts dbconn.TxPreparedStmts) (err error) {
		defer recoverFromError(&err, tr)

		if tr[tracking.ResultStatusKey].Int64() == int64(parser.ExpiredStatus) {
			if err := setQueueExpired(tr[tracking.QueueDeliveryNameKey].Text(), tr[tracking.MessageExpiredTime].Int64(), tx, stmts); err != nil {
				return errorutil.Wrap(err)
			}

			return nil
		}

		if err = handleNonExpiredDeliveryAttempt(tr, tx, stmts); err != nil {
			return errorutil.Wrap(err)
		}

		return nil
	}
}

func handleNonExpiredDeliveryAttempt(tr tracking.Result, tx *sql.Tx, stmts dbconn.TxPreparedStmts) error {
	result, err := insertMandatoryResultFields(tx, stmts, tr)
	if err != nil {
		return errorutil.Wrap(err)
	}

	port := func() int64 {
		if tr[tracking.ResultRelayPortKey].IsNone() {
			return 0
		}

		return tr[tracking.ResultRelayPortKey].Int64()
	}()

	rowId, err := result.LastInsertId()
	if err != nil {
		return errorutil.Wrap(err)
	}

	if err = handleQueueInfo(rowId, tr, tx, stmts); err != nil {
		return errorutil.Wrap(err)
	}

	relayId, relayIdFound, err := getOptionalNextRelayId(tx, stmts, tr[tracking.ResultRelayNameKey], tr[tracking.ResultRelayIPKey], port)
	if err != nil {
		return errorutil.Wrap(err)
	}

	if relayIdFound {
		//nolint:sqlclosecheck
		_, err = stmts.Get(updateDeliveryWithRelay).Exec(relayId, rowId)
		if err != nil {
			return errorutil.Wrap(err)
		}
	}

	origRecipientDomainPartId, origRecipientDomainPartFound, err := getOptionalUniqueRemoteDomainNameId(tx, stmts, tr[tracking.ResultOrigRecipientDomainPartKey])
	if err != nil {
		return errorutil.Wrap(err)
	}

	if origRecipientDomainPartFound {
		//nolint:sqlclosecheck
		_, err = stmts.Get(updateDeliveryWithOrigRecipient).Exec(origRecipientDomainPartId, rowId)
		if err != nil {
			return errorutil.Wrap(err)
		}
	}

	return nil
}

func (p *resultsPublisher) Publish(r tracking.Result) {
	p.dbActions <- buildAction(r)
}

func (db *DB) ResultsPublisher() tracking.ResultPublisher {
	return &resultsPublisher{dbActions: db.Actions}
}

func (db *DB) HasLogs() bool {
	conn, release := db.connPair.RoConnPool.Acquire()

	defer release()

	var count int
	err := conn.QueryRow(`select count(*) from deliveries`).Scan(&count)
	errorutil.MustSucceed(err)

	return count > 0
}

func (db *DB) MostRecentLogTime() (time.Time, error) {
	conn, release := db.connPair.RoConnPool.Acquire()

	defer release()

	var ts int64

	err := conn.QueryRow(`select delivery_ts from deliveries order by rowid desc limit 1`).Scan(&ts)

	if err != nil && errors.Is(err, sql.ErrNoRows) {
		return time.Time{}, nil
	}

	if err != nil {
		return time.Time{}, errorutil.Wrap(err)
	}

	return time.Unix(ts, 0).In(time.UTC), nil
}
