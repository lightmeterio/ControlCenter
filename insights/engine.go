package insights

import (
	"database/sql"
	"gitlab.com/lightmeter/controlcenter/insights/core"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/notification"
	"gitlab.com/lightmeter/controlcenter/util"
	"path"
	"time"
)

type txAction func(*sql.Tx) error

type Engine struct {
	core              *core.Core
	insightsStateConn dbconn.ConnPair
	txActions         chan txAction
	fetcher           core.Fetcher
}

func NewCustomEngine(
	workspaceDir string,
	notificationCenter notification.Center,
	options core.Options,
	buildDetectors func(*creator, core.Options) []core.Detector,
) (*Engine, error) {
	stateConn, err := dbconn.NewConnPair(path.Join(workspaceDir, "insights.db"))

	if err != nil {
		return nil, util.WrapError(err)
	}

	defer func() {
		if err != nil {
			util.MustSucceed(stateConn.Close(), "")
		}
	}()

	if err != nil {
		return nil, util.WrapError(err)
	}

	creator, err := newCreator(stateConn.RwConn, notificationCenter)

	if err != nil {
		return nil, util.WrapError(err)
	}

	detectors := buildDetectors(creator, options)

	if err := setupDetectors(detectors, stateConn.RwConn); err != nil {
		return nil, util.WrapError(err)
	}

	c, err := core.New(detectors)

	if err != nil {
		return nil, util.WrapError(err)
	}

	fetcher, err := newFetcher(stateConn.RoConn)

	if err != nil {
		return nil, util.WrapError(err)
	}

	return &Engine{
		core:              c,
		insightsStateConn: stateConn,
		txActions:         make(chan txAction, 1024),
		fetcher:           fetcher,
	}, nil
}

func setupDetectors(detectors []core.Detector, conn dbconn.RwConn) error {
	tx, err := conn.Begin()

	if err != nil {
		return util.WrapError(err)
	}

	defer func() {
		if err != nil {
			util.MustSucceed(tx.Rollback(), "")
		}
	}()

	err = core.SetupAuxTables(tx)

	if err != nil {
		return util.WrapError(err)
	}

	for _, d := range detectors {
		err = d.Setup(tx)

		if err != nil {
			return util.WrapError(err)
		}
	}

	err = tx.Commit()

	if err != nil {
		return util.WrapError(err)
	}

	return nil
}

func (e *Engine) Close() error {
	if err := e.core.Close(); err != nil {
		return util.WrapError(err)
	}

	if err := e.fetcher.Close(); err != nil {
		return util.WrapError(err)
	}

	return nil
}

func spawnInsightsJob(clock core.Clock, e *Engine, cancel <-chan struct{}) {
	for {
		select {
		case <-cancel:
			return
		default:
			execOnSteppers(e.txActions, e.core.Steppers, clock)
			clock.Sleep(time.Second * 2)
		}
	}
}

func execOnSteppers(txActions chan<- txAction, steppers []core.Stepper, clock core.Clock) {
	txActions <- func(tx *sql.Tx) error {
		for _, s := range steppers {
			if err := s.Step(clock, tx); err != nil {
				return util.WrapError(err)
			}
		}

		return nil
	}
}

// whether a new cycle is possible or the execution should finish
func engineCycle(e *Engine) (bool, error) {
	tx, err := e.insightsStateConn.RwConn.Begin()

	if err != nil {
		return false, util.WrapError(err)
	}

	defer func() {
		if err != nil {
			util.MustSucceed(tx.Rollback(), "")
		}
	}()

	action, ok := <-e.txActions

	if !ok {
		return false, nil
	}

	err = action(tx)

	if err != nil {
		return false, util.WrapError(err)
	}

	err = tx.Commit()

	if err != nil {
		return false, util.WrapError(err)
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

func runDatabaseWriterLoop(e *Engine) error {
	// one thread, owning access to the database
	// waits for write actions, like new insights or actions for the user
	// those actions act on a transaction
	for {
		shouldContinue, err := engineCycle(e)

		if err != nil {
			return util.WrapError(err)
		}

		if !shouldContinue {
			return nil
		}
	}
}

func (e *Engine) Run() (func(), func()) {
	clock := &realClock{}

	cancelInsightsJob := make(chan struct{})

	// start generating insights
	go spawnInsightsJob(clock, e, cancelInsightsJob)

	cancelRun := make(chan struct{})
	doneRun := make(chan struct{})

	go func() {
		<-cancelRun
		cancelInsightsJob <- struct{}{}
		close(e.txActions)
	}()

	// TODO: start user actions thread
	// something that reads user actions (resolve insights, etc.)

	go func() {
		util.MustSucceed(runDatabaseWriterLoop(e), "")
		doneRun <- struct{}{}
	}()

	return func() {
			<-doneRun
		}, func() {
			cancelRun <- struct{}{}
		}
}

func (e *Engine) Fetcher() core.Fetcher {
	return e.fetcher
}
