package logdb

import (
	"database/sql"
	"errors"
	"gitlab.com/lightmeter/controlcenter/logdb/migrations"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"

	"gitlab.com/lightmeter/controlcenter/data"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	parser "gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser"
)

func init() {
	// todo if we need to register many payload handler then add support for []payloadHandler to registerPayloadHandler
	registerPayloadHandler(payloadHandler{
		filename:       "1_postfix_payload_smtp.go",
		up:             migrations.UpTableCreationForSmtpSentStatus,
		down:           migrations.DownTableCreationForSmtpSentStatus,
		database:       "logs",
		counter:        countLogsForSmtpSentStatus,
		lastTimeReader: lastTimeInTableReaderForSmtpSentStatus,
	})
}

var ErrCouldNotObtainTimeFromDatabase = errors.New("Could not obtain time from database")

func lastTimeInTableReaderForSmtpSentStatus(db dbconn.RoConn) (int64, error) {
	var v int64

	// FIXME: this query is way too complicated for something so simple
	err := db.QueryRow(`
	select
		read_ts_sec
	from
		postfix_smtp_message_status
	where
		rowid = (select max(rowid) from postfix_smtp_message_status)`).Scan(&v)

	if err != nil {
		return 0, errorutil.Wrap(err)
	}

	return v, nil
}

func countLogsForSmtpSentStatus(db dbconn.RoConn) int {
	value := 0
	errorutil.MustSucceed(db.QueryRow(`select count(*) from postfix_smtp_message_status`).Scan(&value))

	return value
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
		return errorutil.Wrap(err)
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
		return errorutil.Wrap(err)
	}

	return nil
}
