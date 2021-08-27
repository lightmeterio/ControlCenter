// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package dbrunner

import (
	"database/sql"
	"github.com/rs/zerolog/log"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/pkg/runner"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"path"
	"time"
)

type PreparedStmts []*sql.Stmt

func (stmts PreparedStmts) Close() error {
	for _, stmt := range stmts {
		if err := stmt.Close(); err != nil {
			return errorutil.Wrap(err)
		}
	}

	return nil
}

type Action func(*sql.Tx, PreparedStmts) error

type StmtsText map[uint]string

type Runner struct {
	runner.CancellableRunner
	stmts   PreparedStmts
	Actions chan Action
}

func New(timeout time.Duration, actionSize uint, connPair *dbconn.PooledPair, stmts PreparedStmts) Runner {
	actions := make(chan Action, actionSize)

	return Runner{
		stmts:   stmts,
		Actions: actions,
		CancellableRunner: runner.NewCancellableRunner(func(done runner.DoneChan, cancel runner.CancelChan) {
			go func() {
				<-cancel
				close(actions)
			}()

			go func() {
				done <- func() error {
					if err := fillDatabase(timeout, connPair.RwConn, stmts, actions, path.Base(connPair.Filename)); err != nil {
						return errorutil.Wrap(err)
					}

					return nil
				}()
			}()
		}),
	}
}

func PrepareRwStmts(stmtsText StmtsText, conn dbconn.RwConn, stmts PreparedStmts) error {
	for k, v := range stmtsText {
		//nolint:sqlclosecheck
		stmt, err := conn.Prepare(v)
		if err != nil {
			return errorutil.Wrap(err)
		}

		stmts[k] = stmt
	}

	return nil
}

func fillDatabase(timeout time.Duration, conn dbconn.RwConn, stmts PreparedStmts, dbActions <-chan Action, filename string) error {
	var (
		tx                  *sql.Tx = nil
		countPerTransaction int64
	)

	startTransaction := func() error {
		var err error
		if tx, err = conn.Begin(); err != nil {
			return errorutil.Wrap(err, filename)
		}

		return nil
	}

	closeTransaction := func() error {
		// no transaction to commit
		if tx == nil {
			return nil
		}

		// NOTE: improve it to be used for benchmarking
		log.Info().Msgf("Inserted %d rows in a transaction on database %s", countPerTransaction, filename)

		if err := tx.Commit(); err != nil {
			return errorutil.Wrap(err, filename)
		}

		countPerTransaction = 0
		tx = nil

		return nil
	}

	tryToDoAction := func(action Action) error {
		if tx == nil {
			if err := startTransaction(); err != nil {
				return errorutil.Wrap(err, filename)
			}
		}

		if err := action(tx, stmts); err != nil {
			return errorutil.Wrap(err, filename)
		}

		countPerTransaction++

		return nil
	}

	ticker := time.NewTicker(timeout)

	for {
		select {
		case <-ticker.C:
			if err := closeTransaction(); err != nil {
				return errorutil.Wrap(err, filename)
			}
		case action, ok := <-dbActions:
			{
				if !ok {
					log.Info().Msg("Committing because there's nothing left")

					// cancel() has been called!!!
					if err := closeTransaction(); err != nil {
						return errorutil.Wrap(err, filename)
					}

					return nil
				}

				if err := tryToDoAction(action); err != nil {
					return errorutil.Wrap(err, filename)
				}
			}
		}
	}
}
