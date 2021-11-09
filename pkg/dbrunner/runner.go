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
	"time"
)

type Action func(*sql.Tx, dbconn.TxPreparedStmts) error

type Runner struct {
	runner.CancellableRunner
	stmts   dbconn.PreparedStmts
	Actions chan Action
}

func newCleaner(cleanInterval time.Duration, cleaner Action, actions chan<- Action) runner.CancellableRunner {
	return runner.NewCancellableRunner(func(done runner.DoneChan, cancel runner.CancelChan) {
		go func() {
			ticker := time.NewTicker(cleanInterval)

			for {
				select {
				case <-cancel:
					done <- nil
					return
				case <-ticker.C:
					log.Debug().Msgf("Executing database cleaning action")
					actions <- cleaner
				}
			}
		}()
	})
}

func New(timeout time.Duration, actionSize uint, conn dbconn.RwConn, stmts dbconn.PreparedStmts, cleanInterval time.Duration, cleaner Action) Runner {
	actions := make(chan Action, actionSize)

	return Runner{
		stmts:   stmts,
		Actions: actions,
		CancellableRunner: runner.NewCancellableRunner(func(done runner.DoneChan, cancel runner.CancelChan) {
			cleanerDone, cleanerCancel := runner.Run(newCleaner(cleanInterval, cleaner, actions))

			go func() {
				<-cancel
				cleanerCancel()
				_ = cleanerDone()
				close(actions)
			}()

			go func() {
				done <- func() error {
					if err := fillDatabase(timeout, conn, stmts, actions); err != nil {
						return errorutil.Wrap(err)
					}

					return nil
				}()
			}()
		}),
	}
}

// TODO: I could not find a way to simplify this function,
// so I'll silence the linter complaining about complexity...
//nolint:gocognit
func fillDatabase(timeout time.Duration, conn dbconn.RwConn, stmts dbconn.PreparedStmts, dbActions <-chan Action) error {
	var (
		tx                  *sql.Tx = nil
		countPerTransaction int64
		preparedTxStmts     dbconn.TxPreparedStmts
	)

	startTransaction := func() error {
		var err error
		if tx, err = conn.Begin(); err != nil {
			return errorutil.Wrap(err)
		}

		preparedTxStmts = dbconn.TxStmts(tx, stmts)

		return nil
	}

	closeTransaction := func() error {
		// no transaction to commit
		if tx == nil {
			return nil
		}

		if err := preparedTxStmts.Close(); err != nil {
			return errorutil.Wrap(err)
		}

		// NOTE: improve it to be used for benchmarking
		log.Debug().Msgf("Executed %d statements in a transaction", countPerTransaction)

		if err := tx.Commit(); err != nil {
			return errorutil.Wrap(err)
		}

		countPerTransaction = 0
		tx = nil

		return nil
	}

	tryToDoAction := func(action Action) error {
		if tx == nil {
			if err := startTransaction(); err != nil {
				return errorutil.Wrap(err)
			}
		}

		if err := action(tx, preparedTxStmts); err != nil {
			return errorutil.Wrap(err)
		}

		countPerTransaction++

		return nil
	}

	ticker := time.NewTicker(timeout)

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
