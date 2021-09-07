// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package connectionstats

import (
	"database/sql"
	"errors"
	"github.com/rs/zerolog/log"
	_ "gitlab.com/lightmeter/controlcenter/connectionstats/migrations"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/pkg/dbrunner"
	"gitlab.com/lightmeter/controlcenter/pkg/postfix"
	parser "gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"time"
)

// We keep store in a database all the basic statistics (number and type of smtp commands)
// provided by Postfix on all connections that sent the AUTH command on the ports used by MUAs.
// There is no need to to that on the port 25, as it's used by other MTUs only.

type Command int

func (c Command) MarshalText() ([]byte, error) {
	return []byte(commandAsString(c)), nil
}

const (
	// NOTE: we make the values explicit as they are stored in the database.
	// Changing them is a breaking change!
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
	}

	return UnsupportedCommand, ErrCommandNotSupported
}

type dbAction = dbrunner.Action

type publisher struct {
	actions chan<- dbAction
}

func buildAction(record postfix.Record, payload parser.SmtpdDisconnect) dbAction {
	return func(tx *sql.Tx, stmts dbrunner.PreparedStmts) error {
		stmt := tx.Stmt(stmts[insertDisconnectKey])

		defer stmt.Close()

		r, err := stmt.Exec(record.Time.Unix(), payload.IP)
		if err != nil {
			return errorutil.Wrap(err, record.Location)
		}

		connectionId, err := r.LastInsertId()
		if err != nil {
			return errorutil.Wrap(err)
		}

		stmt = tx.Stmt(stmts[insertCommandStatKey])

		defer stmt.Close()

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

			if _, err := stmt.Exec(connectionId, cmd, v.Success, v.Total); err != nil {
				return errorutil.Wrap(err)
			}
		}

		return nil
	}
}

const (
	insertDisconnectKey = iota
	insertCommandStatKey

	lastStmtKey
)

type preparedStmts [lastStmtKey]*sql.Stmt

var stmtsText = map[uint]string{
	insertDisconnectKey:  `insert into connections(disconnection_ts, ip) values(?, ?)`,
	insertCommandStatKey: `insert into commands(connection_id, cmd, success, total) values(?, ?, ?, ?)`,
}

func (pub *publisher) Publish(r postfix.Record) {
	p, isDisconnect := r.Payload.(parser.SmtpdDisconnect)

	if !isDisconnect {
		return
	}

	if p.IP == nil {
		return
	}

	// NOTE: we want to store statistics of connections that tried, either successfully or not, to authenticate
	if _, ok := p.Stats[commandAsString(AuthCommand)]; ok {
		pub.actions <- buildAction(r, p)
	}
}

type Stats struct {
	dbrunner.Runner

	conn  *dbconn.PooledPair
	stmts preparedStmts
}

func New(connPair *dbconn.PooledPair) (*Stats, error) {
	stmts := preparedStmts{}

	if err := dbrunner.PrepareRwStmts(stmtsText, connPair.RwConn, stmts[:]); err != nil {
		return nil, errorutil.Wrap(err)
	}

	return &Stats{
		conn:   connPair,
		stmts:  stmts,
		Runner: dbrunner.New(500*time.Millisecond, 4096, connPair, stmts[:]),
	}, nil
}

func (s *Stats) Publisher() postfix.Publisher {
	return &publisher{actions: s.Actions}
}

func (s *Stats) ConnPool() *dbconn.RoPool {
	return s.conn.RoConnPool
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
