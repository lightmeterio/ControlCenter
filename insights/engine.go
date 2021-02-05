// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package insights

import (
	"database/sql"
	"gitlab.com/lightmeter/controlcenter/insights/core"
	_ "gitlab.com/lightmeter/controlcenter/insights/migrations"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/migrator"
	"gitlab.com/lightmeter/controlcenter/notification"
	"gitlab.com/lightmeter/controlcenter/pkg/runner"
	"gitlab.com/lightmeter/controlcenter/util/closeutil"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"path"
	"time"
)

type txAction func(*sql.Tx) error

type Engine struct {
	core              *core.Core
	insightsStateConn dbconn.ConnPair
	txActions         chan txAction
	fetcher           core.Fetcher
	closers           closeutil.Closers
	runner.CancelableRunner
}

func NewCustomEngine(
	workspaceDir string,
	notificationCenter notification.Center,
	options core.Options,
	buildDetectors func(*creator, core.Options) []core.Detector,
	additionalActions func([]core.Detector, dbconn.RwConn) error,
) (*Engine, error) {
	stateConn, err := dbconn.NewConnPair(path.Join(workspaceDir, "insights.db"))

	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	defer func() {
		if err != nil {
			errorutil.MustSucceed(stateConn.Close())
		}
	}()

	if err := migrator.Run(stateConn.RwConn.DB, "insights"); err != nil {
		return nil, errorutil.Wrap(err)
	}

	creator, err := newCreator(stateConn.RwConn, notificationCenter)

	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	detectors := buildDetectors(creator, options)

	err = additionalActions(detectors, stateConn.RwConn)

	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	c, err := core.New(detectors)

	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	fetcher, err := newFetcher(stateConn.RoConn)

	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	e := &Engine{
		core:              c,
		insightsStateConn: stateConn,
		txActions:         make(chan txAction, 1024),
		fetcher:           fetcher,
		closers: closeutil.New(
			c,
			fetcher,
		),
	}

	execute := func(done runner.DoneChan, cancel runner.CancelChan) {
		cancelInsightsJob := make(chan struct{})

		clock := &realClock{}
		// start generating insights
		go spawnInsightsJob(clock, e, cancelInsightsJob)

		go func() {
			<-cancel
			cancelInsightsJob <- struct{}{}

			close(e.txActions)
		}()

		go func() {
			runDatabaseWriterLoop(e)
			done <- nil
		}()
	}

	e.CancelableRunner = runner.NewCancelableRunner(execute)

	return e, nil
}

func (e *Engine) Close() error {
	return e.closers.Close()
}

func spawnInsightsJob(clock core.Clock, e *Engine, cancel <-chan struct{}) {
	for {
		select {
		case <-cancel:
			return
		default:
			execOnDetectors(e.txActions, e.core.Detectors, clock)
			clock.Sleep(time.Second * 2)
		}
	}
}

func execOnDetectors(txActions chan<- txAction, steppers []core.Detector, clock core.Clock) {
	txActions <- func(tx *sql.Tx) error {
		for _, s := range steppers {
			if err := s.Step(clock, tx); err != nil {
				return errorutil.Wrap(err)
			}
		}

		return nil
	}
}

// whether a new cycle is possible or the execution should finish
func engineCycle(e *Engine) (bool, error) {
	tx, err := e.insightsStateConn.RwConn.Begin()

	if err != nil {
		return false, errorutil.Wrap(err)
	}

	defer func() {
		if err != nil {
			errorutil.MustSucceed(tx.Rollback())
		}
	}()

	action, ok := <-e.txActions

	if !ok {
		return false, nil
	}

	err = action(tx)

	if err != nil {
		return false, errorutil.Wrap(err)
	}

	err = tx.Commit()

	if err != nil {
		return false, errorutil.Wrap(err)
	}

	return true, nil
}

type realClock struct{}

func (realClock) Now() time.Time {
	return time.Now()
}

func (realClock) Sleep(d time.Duration) {
	time.Sleep(d)
}

func runDatabaseWriterLoop(e *Engine) {
	// one thread, owning access to the database
	// waits for write actions, like new insights or actions for the user
	// those actions act on a transaction
	for {
		shouldContinue, err := engineCycle(e)

		if err != nil {
			errorutil.LogErrorf(err, "Could not not run Insights Engine cycle")
			continue
		}

		if !shouldContinue {
			return
		}
	}
}

func (e *Engine) Fetcher() core.Fetcher {
	return e.fetcher
}
