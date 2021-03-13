// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package insights

import (
	"context"
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

type Accessor struct {
	closeutil.Closers
	conn *dbconn.PooledPair
}

func NewAccessor(workspaceDir string) (*Accessor, error) {
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

	return &Accessor{conn: stateConn, Closers: closeutil.New(stateConn)}, nil
}

func (c *Accessor) NotificationPolicy() notification.Policy {
	return notification.Policies{&doNotGenerateNotificationsDuringImportPolicy{pool: c.conn.RoConnPool}, &DefaultNotificationPolicy{}}
}

type Engine struct {
	runner.CancelableRunner
	accessor        *Accessor
	core            *core.Core
	txActions       chan txAction
	fetcher         core.Fetcher
	closers         closeutil.Closers
	importAnnouncer importAnnouncer
	progressFetcher core.ProgressFetcher
}

func NewCustomEngine(
	c *Accessor,
	notificationCenter *notification.Center,
	options core.Options,
	buildDetectors func(*creator, core.Options) []core.Detector,
	additionalActions func([]core.Detector, dbconn.RwConn, core.Clock) error,
) (*Engine, error) {
	creator, err := newCreator(c.conn.RwConn, notificationCenter)
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	progressFetcher, err := core.NewProgressFetcher(c.conn.RoConnPool)
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	detectors := buildDetectors(creator, options)

	core, err := core.New(detectors)
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	fetcher, err := newFetcher(c.conn.RoConnPool)
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	announcer := importAnnouncer{
		start:    make(chan time.Time),
		progress: make(chan announcer.Progress, 100),
	}

	e := &Engine{
		accessor:        c,
		core:            core,
		txActions:       make(chan txAction, 1024),
		fetcher:         fetcher,
		closers:         closeutil.New(c, core),
		importAnnouncer: announcer,
		progressFetcher: progressFetcher,
	}

	execute := func(done runner.DoneChan, cancel runner.CancelChan) {
		cancelInsightsJob := make(chan struct{})

		// start generating insights
		go spawnInsightsJob(&realClock{}, e, cancelInsightsJob)

		go func() {
			<-cancel
			cancelInsightsJob <- struct{}{}

			close(e.txActions)
		}()

		go func() {
			err := runOnHistoricalData(e)
			errorutil.MustSucceed(err)
			err = additionalActions(detectors, c.conn.RwConn, &realClock{})
			errorutil.MustSucceed(err)
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
func engineCycle(e *Engine) (shouldContinue bool, err error) {
	tx, err := e.accessor.conn.RwConn.Begin()

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

type doNotGenerateNotificationsDuringImportPolicy struct {
	pool *dbconn.RoPool
}

func (p *doNotGenerateNotificationsDuringImportPolicy) Reject(n notification.Notification) (bool, error) {
	running, err := core.IsHistoricalImportRunningFromPool(context.Background(), p.pool)
	if err != nil {
		return false, errorutil.Wrap(err)
	}

	return running, nil
}

func runOnHistoricalData(e *Engine) error {
	interval, err := importHistoricalInsights(e)
	if err != nil {
		return errorutil.Wrap(err)
	}

	// in case we skip the import
	if interval.IsZero() {
		return nil
	}

	if err := generateImportSummaryInsight(e, interval); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func importHistoricalInsights(e *Engine) (timeutil.TimeInterval, error) {
	{
		tx, err := e.accessor.conn.RwConn.Begin()
		if err != nil {
			return timeutil.TimeInterval{}, errorutil.Wrap(err)
		}

		if err := core.EnableHistoricalImportFlag(context.Background(), tx); err != nil {
			return timeutil.TimeInterval{}, errorutil.Wrap(err)
		}

		if err := tx.Commit(); err != nil {
			return timeutil.TimeInterval{}, errorutil.Wrap(err)
		}
	}

	log.Info().Msg("Waiting for import announcement!")

	importStartTime := time.Now()

	start := <-e.importAnnouncer.start

	finish := start

	log.Info().Msgf("Starting insights on historical data starting with %v", start)

	clock := historicalClock{current: start}

	historicalDetectors := []core.HistoricalDetector{}

	for _, s := range e.core.Detectors {
		if h, ok := s.(core.HistoricalDetector); ok {
			historicalDetectors = append(historicalDetectors, h)
		}
	}

	for progress := range e.importAnnouncer.progress {
		tx, err := e.accessor.conn.RwConn.Begin()
		if err != nil {
			return timeutil.TimeInterval{}, errorutil.Wrap(err)
		}

		log.Info().Msgf("Before: Notifying historical import progress of %v%% at %v", progress.Progress, clock.Now())

		for clock.current.Before(progress.Time) {
			for _, h := range historicalDetectors {
				if err := h.Step(&clock, tx); err != nil {
					return timeutil.TimeInterval{}, errorutil.Wrap(err)
				}
			}

			clock.Sleep(time.Minute * 20)
		}

		log.Info().Msgf("After: Notifying historical import progress of %v%% at %v", progress.Progress, clock.Now())

		if progress.Finished {
			finish = progress.Time
			log.Info().Msgf("Finished importing historical data in the time %v", progress.Time)
		}

		if _, err := tx.Exec(`insert into import_progress(value, timestamp, exec_timestamp) values(?, ?, ?)`, progress.Progress, progress.Time.Unix(), time.Now().Unix()); err != nil {
			return timeutil.TimeInterval{}, errorutil.Wrap(err)
		}

		if err := tx.Commit(); err != nil {
			return timeutil.TimeInterval{}, errorutil.Wrap(err)
		}
	}

	{
		tx, err := e.accessor.conn.RwConn.Begin()
		if err != nil {
			return timeutil.TimeInterval{}, errorutil.Wrap(err)
		}

		if err := core.DisableHistoricalImportFlag(context.Background(), tx); err != nil {
			return timeutil.TimeInterval{}, errorutil.Wrap(err)
		}

		// Prevents any non historical insight of being "poisoned" by historical insights
		// It deletes the traces of executions of previous insights
		// NOTE: this is a very ad-hoc and ugly solution, as we might have more tables in the future
		// with data used while insights are being created
		if _, err := tx.Exec(`delete from last_detector_execution`); err != nil {
			return timeutil.TimeInterval{}, errorutil.Wrap(err)
		}

		if err := tx.Commit(); err != nil {
			return timeutil.TimeInterval{}, errorutil.Wrap(err)
		}
	}

	log.Debug().Msgf("Importing historical insights from %v to %v took %v", start, finish, time.Since(importStartTime))

	return timeutil.TimeInterval{From: start, To: finish}, nil
}

func generateImportSummaryInsight(e *Engine, interval timeutil.TimeInterval) error {
	tx, err := e.accessor.conn.RwConn.Begin()
	if err != nil {
		return errorutil.Wrap(err)
	}

	// Single shot detector
	summaryInsightDetector := importsummary.NewDetector(e.fetcher, interval)

	// Generate an import summary insight
	if err := summaryInsightDetector.Step(&timeutil.RealClock{}, tx); err != nil {
		return errorutil.Wrap(err)
	}

	if err := tx.Commit(); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func (e *Engine) Fetcher() core.Fetcher {
	return e.fetcher
}

func (e *Engine) ImportAnnouncer() announcer.ImportAnnouncer {
	return &e.importAnnouncer
}

func (e *Engine) ProgressFetcher() core.ProgressFetcher {
	return e.progressFetcher
}
