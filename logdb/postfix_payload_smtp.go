package logdb

import (
	"database/sql"
	"errors"

	"gitlab.com/lightmeter/controlcenter/data"
	"gitlab.com/lightmeter/controlcenter/util"
	parser "gitlab.com/lightmeter/postfix-log-parser"
)

func init() {
	registerPayloadHandler(payloadHandler{
		creator:        tableCreationForSmtpSentStatus,
		counter:        countLogsForSmtpSentStatus,
		lastTimeReader: lastTimeInTableReaderForSmtpSentStatus,
	})
}

var ErrCouldNotObtainTimeFromDatabase = errors.New("Could not obtain time from database")

func lastTimeInTableReaderForSmtpSentStatus(db *sql.DB) (int64, error) {
	// FIXME: this query is way too complicated for something so simple

	var v int64

	err := db.QueryRow(`
	select
		read_ts_sec
	from
		postfix_smtp_message_status
	where
		rowid = (select max(rowid) from postfix_smtp_message_status)`).Scan(&v)

	if err != nil {
		return 0, util.WrapError(err)
	}

	return v, nil
}

func countLogsForSmtpSentStatus(db *sql.DB) int {
	value := 0
	util.MustSucceed(db.QueryRow(`select count(*) from postfix_smtp_message_status`).Scan(&value), "")
	return value
}

func tableCreationForSmtpSentStatus(db *sql.DB) error {
	if _, err := db.Exec(`create table if not exists postfix_smtp_message_status(
	read_ts_sec           integer,
	process_ip            blob,
	queue                 string,
	recipient_local_part  text,
	recipient_domain_part text,
	relay_name            text,
	relay_ip              blob,
	relay_port            uint16,
	delay                 double,
	delay_smtpd           double,
	delay_cleanup         double,
	delay_qmgr            double,
	delay_smtp            double,
	dsn                   text,
	status                integer
		)`); err != nil {
		return util.WrapError(err)
	}

	if _, err := db.Exec(`create index if not exists postfix_smtp_message_time_index
		on postfix_smtp_message_status (read_ts_sec)`); err != nil {
		return util.WrapError(err)
	}

	return nil
}

func inserterForSmtpSentStatus(tx *sql.Tx, r data.Record) error {
	status, _ := r.Payload.(parser.SmtpSentStatus)

	stmt, err := tx.Prepare(`
		insert into postfix_smtp_message_status(
			read_ts_sec,
			process_ip,
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
		) values(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)

	if err != nil {
		return util.WrapError(err)
	}

	defer stmt.Close()

	_, err = stmt.Exec(
		r.Time.Unix(),
		r.Header.ProcessIP,
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
		return util.WrapError(err)
	}

	return nil
}
