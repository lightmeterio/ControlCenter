// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package insights

import (
	"database/sql"
	"github.com/rs/zerolog/log"
	"gitlab.com/lightmeter/controlcenter/insights/core"
	"gitlab.com/lightmeter/controlcenter/insights/importsummary"
	_ "gitlab.com/lightmeter/controlcenter/insights/migrations"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/migrator"
	"gitlab.com/lightmeter/controlcenter/logeater/announcer"
	"gitlab.com/lightmeter/controlcenter/notification"
	"gitlab.com/lightmeter/controlcenter/pkg/runner"
	"gitlab.com/lightmeter/controlcenter/util/closeutil"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"path"
	"time"
)

type txAction func(*sql.Tx) error

type importAnnouncer struct {
	start    chan time.Time
	progress chan announcer.Progress
}

func (a *importAnnouncer) AnnounceStart(time time.Time) {
	a.start <- time
}

func (a *importAnnouncer) AnnounceProgress(p announcer.Progress) {
	a.progress <- p

	if p.Finished {
		close(a.progress)
	}
}

type Engine struct {
	core              *core.Core
	insightsStateConn *dbconn.PooledPair
	txActions         chan txAction
	fetcher           core.Fetcher
	closers           closeutil.Closers
	runner.CancelableRunner
	importAnnouncer    importAnnouncer
	notificationCenter *notification.Center
}

func NewCustomEngine(
	workspaceDir string,
	notificationCenter *notification.Center,
	options core.Options,
	buildDetectors func(*creator, core.Options) []core.Detector,
	additionalActions func([]core.Detector, dbconn.RwConn) error,
) (*Engine, error) {
	stateConn, err := dbconn.Open(path.Join(workspaceDir, "insights.db"), 10)

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

	fetcher, err := newFetcher(stateConn.RoConnPool)

	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	announcer := importAnnouncer{
		start:    make(chan time.Time),
		progress: make(chan announcer.Progress, 100),
	}

	e := &Engine{
		core:               c,
		insightsStateConn:  stateConn,
		txActions:          make(chan txAction, 1024),
		fetcher:            fetcher,
		closers:            closeutil.New(c),
		importAnnouncer:    announcer,
		notificationCenter: notificationCenter,
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
	if err := e.core.Close(); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
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

type realClock = timeutil.RealClock

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

type historicalClock struct {
	current time.Time
}

func (c *historicalClock) Now() time.Time {
	return c.current
}

func (c *historicalClock) Sleep(d time.Duration) {
	c.current = c.current.Add(d)
}

// TODO: error handling here!!!
func runOnHistoricalData(e *Engine) {
	interval := importHistoricalInsights(e)

	if !interval.To.IsZero() {
		generateImportSummaryInsight(e, interval)
	}
}

func importHistoricalInsights(e *Engine) timeutil.TimeInterval {
	log.Info().Msg("Waiting for import announcement!")

	importStartTime := time.Now()

	start := <-e.importAnnouncer.start

	log.Info().Msgf("Starting insights on historical data starting with %v", start)

	clock := historicalClock{current: start}

	tx, err := e.insightsStateConn.RwConn.Begin()
	errorutil.MustSucceed(err)

	historicalDetectors := []core.HistoricalDetector{}

	for _, s := range e.core.Detectors {
		if h, ok := s.(core.HistoricalDetector); ok {
			historicalDetectors = append(historicalDetectors, h)
		}
	}

	finish := start

	for progress := range e.importAnnouncer.progress {
		log.Info().Msgf("Before: Notifying historical import progress of %v%% at %v", progress.Progress, clock.Now())

		for clock.current.Before(progress.Time) {
			for _, h := range historicalDetectors {
				errorutil.MustSucceed(h.Step(&clock, tx))
			}

			clock.Sleep(time.Minute * 20)
		}

		log.Info().Msgf("After: Notifying historical import progress of %v%% at %v", progress.Progress, clock.Now())

		if progress.Finished {
			finish = progress.Time
			log.Info().Msgf("Finished importing historical data in the time %v", progress.Time)
		}
	}

	errorutil.MustSucceed(tx.Commit())

	log.Debug().Msgf("Importing historical insights from %v to %v took %v", start, finish, time.Since(importStartTime))

	return timeutil.TimeInterval{From: start, To: finish}
}

func generateImportSummaryInsight(e *Engine, interval timeutil.TimeInterval) {
	tx, err := e.insightsStateConn.RwConn.Begin()
	errorutil.MustSucceed(err)

	// Single shot detector
	summaryInsightDetector := importsummary.NewDetector(e.fetcher, interval)

	// Generate an import summary insight
	errorutil.MustSucceed(summaryInsightDetector.Step(&timeutil.RealClock{}, tx))

	errorutil.MustSucceed(tx.Commit())
}

// TODO: turn this into a runner.CancellableRunner!!!
func (e *Engine) Run() (func() error, func()) {
	doneRun := make(chan error)

	go func() {
		runOnHistoricalData(e)
		runDatabaseWriterLoop(e)
		doneRun <- nil
	}()

	cancelInsightsJob := make(chan struct{})

	// start generating insights
	go spawnInsightsJob(&realClock{}, e, cancelInsightsJob)

	cancelRun := make(chan struct{})

	go func() {
		<-cancelRun
		cancelInsightsJob <- struct{}{}

		close(e.txActions)
	}()

	// TODO: start user actions thread
	// something that reads user actions (resolve insights, etc.)

	return func() error {
			return <-doneRun
		}, func() {
			cancelRun <- struct{}{}
		}
}

func (e *Engine) Fetcher() core.Fetcher {
	return e.fetcher
}

func (e *Engine) ImportAnnouncer() announcer.ImportAnnouncer {
	return &e.importAnnouncer
}
