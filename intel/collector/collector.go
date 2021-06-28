// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package collector

import (
	_ "gitlab.com/lightmeter/controlcenter/intel/migrations"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/migrator"
	"gitlab.com/lightmeter/controlcenter/meta"
	"gitlab.com/lightmeter/controlcenter/pkg/runner"
	"gitlab.com/lightmeter/controlcenter/util/closeutil"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"

	"github.com/rs/zerolog/log"

	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"path"
	"time"
)

func Collect(tx *sql.Tx, clock timeutil.Clock, id string, report ReportPayload) error {
	j, err := json.Marshal(report)
	if err != nil {
		return errorutil.Wrap(err)
	}

	now := clock.Now()

	log.Info().Msgf("Collecting report with id %s at %v", id, now)

	if _, err := tx.Exec(`insert into queued_reports(time, identifier, value) values(?, ?, ?)`, now.Unix(), id, j); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

type Reporters []Reporter

func (reporters Reporters) Step(tx *sql.Tx, clock timeutil.Clock) error {
	for _, r := range reporters {
		lastExecTime, err := func() (time.Time, error) {
			var lastExecTs int64

			err := meta.Retrieve(context.Background(), tx, r.ID(), &lastExecTs)

			// first execution. Not an error
			if err != nil && errors.Is(err, meta.ErrNoSuchKey) {
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
			if err := meta.Store(context.Background(), tx, []meta.Item{{Key: r.ID(), Value: now.Unix()}}); err != nil {
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
	runner.CancelableRunner
	closeutil.Closers

	reporters Reporters
}

type Options struct {
	// How often should the c
	CycleInterval time.Duration

	// How often should the reports be dispatched/sent?
	ReportInterval time.Duration
}

func New(workspace string, options Options, reporters Reporters, dispatcher Dispatcher) (*Collector, error) {
	return NewWithCustomClock(workspace, options, reporters, dispatcher, &timeutil.RealClock{})
}

// NOTE: New takes ownwership of the reporters, calling Close() when it ends
func NewWithCustomClock(workspace string, options Options, reporters Reporters, dispatcher Dispatcher, clock timeutil.Clock) (*Collector, error) {
	pair, err := dbconn.Open(path.Join(workspace, "intel-collector.db"), 4)
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	if err := migrator.Run(pair.RwConn.DB, "intel"); err != nil {
		return nil, errorutil.Wrap(err)
	}

	closers := closeutil.New(pair)

	for _, r := range reporters {
		closers.Add(r)
	}

	return &Collector{
		reporters: reporters,
		Closers:   closers,
		CancelableRunner: runner.NewCancelableRunner(func(done runner.DoneChan, cancel runner.CancelChan) {
			go func() {
				timer := time.NewTicker(options.CycleInterval)

				for {
					select {
					case <-cancel:
						log.Info().Msgf("Intel collector asked to stop at %v!", clock.Now())

						done <- nil

						timer.Stop()

						return
					case <-timer.C:
						if err := pair.RwConn.Tx(func(tx *sql.Tx) error {
							if err := Step(tx, clock, reporters, dispatcher, options.ReportInterval); err != nil {
								return errorutil.Wrap(err)
							}

							return nil
						}); err != nil {
							done <- err
							return
						}
					}
				}
			}()
		}),
	}, nil
}
