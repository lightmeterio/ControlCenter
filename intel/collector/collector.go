// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package collector

import (
	_ "gitlab.com/lightmeter/controlcenter/intel/migrations"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/metadata"
	"gitlab.com/lightmeter/controlcenter/pkg/dbrunner"
	"gitlab.com/lightmeter/controlcenter/pkg/runner"
	"gitlab.com/lightmeter/controlcenter/util/closeutil"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"

	"github.com/rs/zerolog/log"

	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"
)

func Collect(tx *sql.Tx, clock timeutil.Clock, id string, report ReportPayload) error {
	j, err := json.Marshal(report)
	if err != nil {
		return errorutil.Wrap(err)
	}

	now := clock.Now()

	log.Info().Msgf("Collecting report with id %s at %v", id, now)

	if _, err := tx.Exec(`insert into queued_reports(time, identifier, value, dispatched_time) values(?, ?, ?, 0)`, now.Unix(), id, j); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

type Reporters []Reporter

func (reporters Reporters) Step(tx *sql.Tx, clock timeutil.Clock) error {
	for _, r := range reporters {
		lastExecTime, err := func() (time.Time, error) {
			var lastExecTs int64

			err := metadata.Retrieve(context.Background(), tx, r.ID(), &lastExecTs)

			// first execution. Not an error
			if err != nil && errors.Is(err, metadata.ErrNoSuchKey) {
				return time.Time{}, nil
			}

			if err != nil {
				return time.Time{}, errorutil.Wrap(err)
			}

			time := time.Unix(lastExecTs, 0)

			return time, nil
		}()

		if err != nil {
			return errorutil.Wrap(err, "id:", r.ID())
		}

		now := clock.Now()

		storeLastExec := func() error {
			if err := metadata.Store(context.Background(), tx, []metadata.Item{{Key: r.ID(), Value: now.Unix()}}); err != nil {
				return errorutil.Wrap(err, "id:", r.ID())
			}

			return nil
		}

		if lastExecTime.IsZero() {
			log.Info().Msgf("First exec try for %s. Skipping...", r.ID())

			if err := storeLastExec(); err != nil {
				return errorutil.Wrap(err)
			}

			continue
		}

		executionInterval := r.ExecutionInterval()
		execDiff := now.Sub(lastExecTime)

		if execDiff < executionInterval {
			continue
		}

		log.Info().Msgf("Executing intel collector %s on time %v", r.ID(), now)

		if err := r.Step(tx, clock); err != nil {
			return errorutil.Wrap(err, "id:", r.ID())
		}

		if err := storeLastExec(); err != nil {
			return errorutil.Wrap(err)
		}
	}

	return nil
}

type Collector struct {
	runner.CancellableRunner
	closeutil.Closers

	reporters Reporters
}

type Options struct {
	// How often should the c
	CycleInterval time.Duration

	// How often should the reports be dispatched/sent?
	ReportInterval time.Duration
}

func New(intelDb *dbconn.PooledPair, options Options, reporters Reporters, dispatcher Dispatcher) (*Collector, error) {
	return NewWithCustomClock(intelDb, options, reporters, dispatcher, &timeutil.RealClock{})
}

// NOTE: New takes ownwership of the reporters, calling Close() when it ends
func NewWithCustomClock(pair *dbconn.PooledPair, options Options, reporters Reporters, dispatcher Dispatcher, clock timeutil.Clock) (*Collector, error) {
	closers := closeutil.New()

	for _, r := range reporters {
		closers.Add(r)
	}

	stmts := dbconn.PreparedStmts{}

	// ~3 months. TODO: make it configurable
	const maxAge = (time.Hour * 24 * 30 * 3)

	return &Collector{
		reporters: reporters,
		Closers:   closers,
		CancellableRunner: runner.NewCancellableRunner(func(done runner.DoneChan, cancel runner.CancelChan) {
			dbRunner := dbrunner.New(options.CycleInterval, 10, pair.RwConn, stmts, time.Hour*12, makeCleanAction(maxAge))
			dbRunnerDone, dbRunnerCancel := runner.Run(dbRunner)

			go func() {
				timer := time.NewTicker(options.CycleInterval)

				for {
					select {
					case <-cancel:
						log.Info().Msgf("Intel collector asked to stop at %v!", clock.Now())

						timer.Stop()
						dbRunnerCancel()
						done <- dbRunnerDone()

						return
					case <-timer.C:
						dbRunner.Actions <- func(tx *sql.Tx, _ dbconn.TxPreparedStmts) error {
							if err := Step(tx, clock, reporters, dispatcher, options.ReportInterval); err != nil {
								return errorutil.Wrap(err)
							}

							return nil
						}
					}
				}
			}()
		}),
	}, nil
}

func makeCleanAction(maxAge time.Duration) dbrunner.Action {
	return func(tx *sql.Tx, stmts dbconn.TxPreparedStmts) error {
		var mostRecentDispatchTime int64
		if err := tx.QueryRow(`select time from queued_reports order by id desc limit 1`).Scan(&mostRecentDispatchTime); err != nil {
			return errorutil.Wrap(err)
		}

		mostRecentTime := time.Unix(mostRecentDispatchTime, 0)
		oldestTimeToKeep := mostRecentTime.Add(-maxAge)
		oldestTimeToKeepInTimestamp := oldestTimeToKeep.Unix()

		if _, err := tx.Exec(`delete from queued_reports where time < ?`, oldestTimeToKeepInTimestamp); err != nil {
			return errorutil.Wrap(err)
		}

		if _, err := tx.Exec(`delete from dispatch_times where time < ?`, oldestTimeToKeepInTimestamp); err != nil {
			return errorutil.Wrap(err)
		}

		return nil
	}
}
