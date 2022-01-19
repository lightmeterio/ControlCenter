// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package connectionstats

import (
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/rs/zerolog/log"
	_ "gitlab.com/lightmeter/controlcenter/connectionstats/migrations"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/pkg/dbrunner"
	"gitlab.com/lightmeter/controlcenter/pkg/postfix"
	parser "gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser"
	"gitlab.com/lightmeter/controlcenter/util/closeutil"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"time"
)

// We keep store in a database all the basic statistics (number and type of smtp commands)
// provided by Postfix on all connections that sent the AUTH command on the ports used by MUAs.
// There is no need to to that on the port 25, as it's used by other MTAs only.

type Protocol int

const (
	// NOTE: those values are stored in the database, so please do not change existing ones!
	ProtocolSMTP Protocol = 0
	ProtocolIMAP Protocol = 1
)

type Command int

func (c Command) MarshalText() ([]byte, error) {
	return []byte(commandAsString(c)), nil
}

func (p Protocol) MarshalJSON() ([]byte, error) {
	s := func() string {
		switch p {
		case ProtocolIMAP:
			return "imap"
		case ProtocolSMTP:
			return "smtp"
		default:
			log.Panic().Msgf("Invalid protocol: %#v", p)
			return ""
		}
	}()

	return json.Marshal(s)
}

const (
	// NOTE: we make the values explicit as they are stored in the database.
	// Changing them is a breaking change!

	// postfix/smtp commands from here:
	UnknownCommand  Command = 0
	AuthCommand     Command = 1
	BdatCommand     Command = 2
	DataCommand     Command = 3
	EhloCommand     Command = 4
	HeloCommand     Command = 5
	MailCommand     Command = 6
	QuitCommand     Command = 7
	RcptCommand     Command = 8
	StartTLSCommand Command = 9
	RsetCommand     Command = 10
	NoopCommand     Command = 11
	VrfyCommand     Command = 12
	EtrnCommand     Command = 13
	XclientCommand  Command = 14
	XforwardCommand Command = 15

	// dovecot commands from here

	// those are actually fake commands, as we do
	// not support getting the "raw" IMAP commands
	DovecotAuthCommand  Command = 50
	DovecotBlockCommand Command = 51

	UnsupportedCommand = 999
)

func commandAsString(c Command) string {
	switch c {
	case UnknownCommand:
		return "unknown"
	case AuthCommand:
		return "auth"
	case BdatCommand:
		return "bdat"
	case DataCommand:
		return "data"
	case EhloCommand:
		return "ehlo"
	case HeloCommand:
		return "helo"
	case MailCommand:
		return "mail"
	case QuitCommand:
		return "quit"
	case RcptCommand:
		return "rcpt"
	case StartTLSCommand:
		return "starttls"
	case RsetCommand:
		return "rset"
	case NoopCommand:
		return "noop"
	case VrfyCommand:
		return "vrfy"
	case EtrnCommand:
		return "etrn"
	case XclientCommand:
		return "xclient"
	case XforwardCommand:
		return "xforward"
	case DovecotAuthCommand:
		return "dovecot_auth"
	case DovecotBlockCommand:
		return "dovecot_block"
	}

	return "unsupported"
}

var ErrCommandNotSupported = errors.New(`Command not supported`)

func buildCommand(s string) (Command, error) {
	switch s {
	case "unknown":
		return UnknownCommand, nil
	case "auth":
		return AuthCommand, nil
	case "bdat":
		return BdatCommand, nil
	case "data":
		return DataCommand, nil
	case "ehlo":
		return EhloCommand, nil
	case "helo":
		return HeloCommand, nil
	case "mail":
		return MailCommand, nil
	case "quit":
		return QuitCommand, nil
	case "rcpt":
		return RcptCommand, nil
	case "starttls":
		return StartTLSCommand, nil
	case "rset":
		return RsetCommand, nil
	case "noop":
		return NoopCommand, nil
	case "vrfy":
		return VrfyCommand, nil
	case "etrn":
		return EtrnCommand, nil
	case "xclient":
		return XclientCommand, nil
	case "xforward":
		return XforwardCommand, nil
	case "dovecot_auth":
		return DovecotAuthCommand, nil
	case "dovecot_block":
		return DovecotBlockCommand, nil
	}

	return UnsupportedCommand, ErrCommandNotSupported
}

type dbAction = dbrunner.Action

type publisher struct {
	actions chan<- dbAction
}

func buildSmtpAction(record postfix.Record, payload parser.SmtpdDisconnect) dbAction {
	return func(tx *sql.Tx, stmts dbconn.TxPreparedStmts) error {
		//nolint:sqlclosecheck
		r, err := stmts.Get(insertDisconnectKey).Exec(record.Time.Unix(), payload.IP, ProtocolSMTP)
		if err != nil {
			return errorutil.Wrap(err, record.Location)
		}

		connectionId, err := r.LastInsertId()
		if err != nil {
			return errorutil.Wrap(err)
		}

		for k, v := range payload.Stats {
			// skip useless summary reported by postfix
			if k == "commands" {
				continue
			}

			cmd, err := buildCommand(k)
			if err != nil && errors.Is(err, ErrCommandNotSupported) {
				log.Warn().Msgf("Disconnect stat command not supported: %v", k)
				continue
			}

			//nolint:sqlclosecheck
			if _, err := stmts.Get(insertCommandStatKey).Exec(connectionId, cmd, v.Success, v.Total); err != nil {
				return errorutil.Wrap(err)
			}
		}

		return nil
	}
}

func buildDovecotCommand(p parser.DovecotAuthFailed) Command {
	if p.Reason == parser.DovecotAuthFailedReasonAuthPolicyRefusal {
		return DovecotBlockCommand
	}

	return DovecotAuthCommand
}

func buildDovecotAction(record postfix.Record, payload parser.DovecotAuthFailed) dbAction {
	return func(tx *sql.Tx, stmts dbconn.TxPreparedStmts) error {
		//nolint:sqlclosecheck
		r, err := stmts.Get(insertDisconnectKey).Exec(record.Time.Unix(), payload.IP, ProtocolIMAP)
		if err != nil {
			return errorutil.Wrap(err, record.Location)
		}

		connectionId, err := r.LastInsertId()
		if err != nil {
			return errorutil.Wrap(err)
		}

		//nolint:sqlclosecheck
		if _, err := stmts.Get(insertCommandStatKey).Exec(connectionId, buildDovecotCommand(payload), 0, 1); err != nil {
			return errorutil.Wrap(err)
		}

		return nil
	}
}

const (
	insertDisconnectKey = iota
	insertCommandStatKey
	selectOldLogsKey
	deleteCommandsByConnectionIdKey
	deleteConnectionsByIdKey

	lastStmtKey
)

var stmtsText = map[int]string{
	insertDisconnectKey:  `insert into connections(disconnection_ts, ip, protocol) values(?, ?, ?)`,
	insertCommandStatKey: `insert into commands(connection_id, cmd, success, total) values(?, ?, ?, ?)`,
	selectOldLogsKey: `with time_cut as (
		select
			(disconnection_ts - ?) as v
		from
			connections
		order by
			disconnection_ts desc limit 1
	)
	select
		connections.id
	from
		connections join time_cut
			on connections.disconnection_ts < time_cut.v
	limit ?`,
	deleteCommandsByConnectionIdKey: `delete from commands where connection_id = ?`,
	deleteConnectionsByIdKey:        `delete from connections where id = ?`,
}

func (pub *publisher) Publish(r postfix.Record) {
	switch p := r.Payload.(type) {
	case parser.SmtpdDisconnect:
		if p.IP == nil {
			return
		}

		// NOTE: we want to store statistics of connections that tried, either successfully or not, to authenticate
		if _, ok := p.Stats[commandAsString(AuthCommand)]; ok {
			pub.actions <- buildSmtpAction(r, p)
		}
	case parser.DovecotAuthFailed:
		failed := p.Reason == parser.DovecotAuthFailedReasonUnknownUser ||
			p.Reason == parser.DovecotAuthFailedReasonPasswordMismatch ||
			p.Reason == parser.DovecotAuthFailedReasonAuthPolicyRefusal

		if failed {
			pub.actions <- buildDovecotAction(r, p)
		}
	}
}

type Stats struct {
	*dbrunner.Runner
	closeutil.Closers

	conn *dbconn.PooledPair
}

func New(connPair *dbconn.PooledPair) (*Stats, error) {
	stmts := dbconn.BuildPreparedStmts(lastStmtKey)

	if err := dbconn.PrepareRwStmts(stmtsText, connPair.RwConn, &stmts); err != nil {
		return nil, errorutil.Wrap(err)
	}

	// ~3 months. TODO: make it configurable
	const (
		maxAge            = (time.Hour * 24 * 30 * 3)
		cleaningFrequency = time.Minute * 2
		cleaningBatchSize = 1000
	)

	return &Stats{
		conn:    connPair,
		Runner:  dbrunner.New(500*time.Millisecond, 4096, connPair.RwConn, stmts, cleaningFrequency, makeCleanAction(maxAge, cleaningBatchSize)),
		Closers: closeutil.New(stmts),
	}, nil
}

func (s *Stats) Publisher() postfix.Publisher {
	return &publisher{actions: s.Actions}
}

func (s *Stats) MostRecentLogTime() (time.Time, error) {
	conn, release := s.conn.RoConnPool.Acquire()

	defer release()

	var ts int64

	err := conn.QueryRow(`select disconnection_ts from connections order by id desc limit 1`).Scan(&ts)

	if err != nil && errors.Is(err, sql.ErrNoRows) {
		return time.Time{}, nil
	}

	if err != nil {
		return time.Time{}, errorutil.Wrap(err)
	}

	return time.Unix(ts, 0).In(time.UTC), nil
}

func makeCleanAction(maxAge time.Duration, batchSize int) dbrunner.Action {
	return func(tx *sql.Tx, stmts dbconn.TxPreparedStmts) error {
		// NOTE: timestamp is in seconds
		//nolint:sqlclosecheck
		rows, err := stmts.Get(selectOldLogsKey).Query(maxAge/time.Second, batchSize)
		if err != nil {
			return errorutil.Wrap(err)
		}

		defer rows.Close()

		for rows.Next() {
			var id int64

			if err := rows.Scan(&id); err != nil {
				return errorutil.Wrap(err)
			}

			//nolint:sqlclosecheck
			if _, err := stmts.Get(deleteCommandsByConnectionIdKey).Exec(id); err != nil {
				return errorutil.Wrap(err)
			}

			//nolint:sqlclosecheck
			if _, err := stmts.Get(deleteConnectionsByIdKey).Exec(id); err != nil {
				return errorutil.Wrap(err)
			}
		}

		if err := rows.Err(); err != nil {
			return errorutil.Wrap(err)
		}

		return nil
	}
}
