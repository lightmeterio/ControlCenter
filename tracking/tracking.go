package tracking

import (
	"database/sql"
	"gitlab.com/lightmeter/controlcenter/data"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/migrator"
	parser "gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser"
	"gitlab.com/lightmeter/controlcenter/pkg/runner"
	_ "gitlab.com/lightmeter/controlcenter/tracking/migrations"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"log"
	"path"
	"time"
)

type MessageDirection int

const (
	// NOTE: those values are stored in the database,
	// so changing them must force a data migration to new values!
	MessageDirectionOutbound = 0
	MessageDirectionIncoming = 1
)

/**
 * The tracker keeps state of the postfix actions, and notifies once the destiny of an e-mail is met.
 */

// Every payload type has an action associated to it. The default action is to do nothing.
// An action can use data obtained from the payload itself.

type ActionType int

const (
	UnsupportedActionType ActionType = iota
	UninterestingActionType
	ConnectActionType
	CloneActionType
	CleanupProcessingActionType
	MailQueuedActionType
	DisconnectActionType
	MailSentActionType
	CommitActionType
	MailBouncedActionType
	BounceCreatedActionType
)

type actionTuple struct {
	actionType     ActionType
	record         data.Record
	actionDataPair actionDataPair
}

type actionDataPair struct {
	connectionActionData connectionActionData
	resultActionData     resultActionData
}

type Publisher struct {
	actions chan<- actionTuple
}

func (p *Publisher) Publish(r data.Record) {
	actionType, actionDataPair := actionTypeForRecord(r)

	if actionType != UnsupportedActionType {
		p.actions <- actionTuple{
			actionType:     actionType,
			actionDataPair: actionDataPair,
			record:         r,
		}
	}
}

type actionImpl func(*Tracker, *sql.Tx, data.Record, actionDataPair) error

type actionData func(*Tracker, int64, *sql.Tx, parser.Payload) error

type connectionActionData actionData

type resultActionData actionData

type actionRecord struct {
	impl actionImpl
}

var actions = map[ActionType]actionRecord{
	ConnectActionType:           {impl: connectAction},
	CloneActionType:             {impl: cloneAction},
	CleanupProcessingActionType: {impl: cleanupProcessingAction},
	MailQueuedActionType:        {impl: mailQueuedAction},
	DisconnectActionType:        {impl: disconnectAction},
	MailSentActionType:          {impl: mailSentAction},
	CommitActionType:            {impl: commitAction},
	MailBouncedActionType:       {impl: mailBouncedAction},
	BounceCreatedActionType:     {impl: bounceCreatedAction},
}

type trackerStmts [lastTrackerStmtKey]*sql.Stmt

type Tracker struct {
	runner.CancelableRunner

	stmts trackerStmts

	dbconn  dbconn.ConnPair
	actions chan actionTuple

	queuesToNotify chan resultInfo

	queuesCommitNotifier *queuesCommitNotifier
}

func (t *Tracker) Publisher() *Publisher {
	return &Publisher{actions: t.actions}
}

func (t *Tracker) Close() error {
	if err := t.dbconn.Close(); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func New(workspaceDir string, pub ResultPublisher) (*Tracker, error) {
	conn, err := dbconn.NewConnPair(path.Join(workspaceDir, "logtracker.db"))

	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	defer func() {
		if err != nil {
			// TODO: do not panic here, but return the error to the caller
			errorutil.MustSucceed(conn.Close())
		}
	}()

	err = migrator.Run(conn.RwConn.DB, "logtracker")
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	trackerStmts, err := prepareTrackerRwStmts(conn.RwConn)
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	notifierStmts, err := prepareNotifierRoStmts(conn.RoConn)
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	queuesToNotify := make(chan resultInfo, 1024*20)

	queuesCommitNotifier := &queuesCommitNotifier{
		resultsToNotify: queuesToNotify,
		publisher:       pub,
	}

	tracker := &Tracker{
		stmts:                trackerStmts,
		dbconn:               conn,
		actions:              make(chan actionTuple, 1024*10),
		queuesToNotify:       queuesToNotify,
		queuesCommitNotifier: queuesCommitNotifier,
	}

	queuesCommitNotifier.CancelableRunner = runner.NewCancelableRunner(func(done runner.DoneChan, cancel runner.CancelChan) {
		go func() {
			<-cancel
			close(queuesToNotify)
		}()

		go func() {
			done <- func() error {
				if err := runQueuesCommitNotifier(notifierStmts, queuesCommitNotifier); err != nil {
					return errorutil.Wrap(err)
				}

				return nil
			}()
		}()
	})

	tracker.CancelableRunner = runner.NewCancelableRunner(func(done runner.DoneChan, cancel runner.CancelChan) {
		notifierDone, notifierCancel := queuesCommitNotifier.Run()

		doneWithTrackerLoop := make(chan struct{})

		go func() {
			<-cancel
			close(tracker.actions)
			<-doneWithTrackerLoop
			notifierCancel()
		}()

		go func() {
			trackerError := func() error {
				if err := runTracker(tracker); err != nil {
					return errorutil.Wrap(err)
				}

				return nil
			}()

			doneWithTrackerLoop <- struct{}{}

			notifierError := notifierDone()

			done <- func() error {
				if notifierError != nil {
					log.Panicln("Tracking commits notifier failed first. Ignore main tracker loop error for now:", trackerError)
					return errorutil.Wrap(notifierError)
				}

				if trackerError != nil {
					return errorutil.Wrap(trackerError)
				}

				return nil
			}()
		}()
	})

	return tracker, nil
}

func startTransactionIfNeeded(conn dbconn.RwConn, tx *sql.Tx) (*sql.Tx, error) {
	if tx != nil {
		return tx, nil
	}

	tx, err := conn.Begin()
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	return tx, nil
}

func executeActionInTransaction(conn dbconn.RwConn, tx *sql.Tx, t *Tracker, actionTuple actionTuple) (*sql.Tx, error) {
	var err error

	actionRecord, found := actions[actionTuple.actionType]

	if !found {
		log.Panicln("SPANK SPANK: Invalid/unsupported action!:", actionTuple)
	}

	action := actionRecord.impl
	actionDataPair := actionTuple.actionDataPair

	if tx, err = startTransactionIfNeeded(conn, tx); err != nil {
		return nil, errorutil.Wrap(err)
	}

	if err = action(t, tx, actionTuple.record, actionDataPair); err != nil {
		return nil, errorutil.Wrap(err, "log file: ", actionTuple.record.Location.Filename, ":", actionTuple.record.Location.Line)
	}

	return tx, nil
}

func runTracker(t *Tracker) error {
	var (
		tx *sql.Tx
	)

	commitTransactionIfNeeded := func() error {
		if tx == nil {
			return nil
		}

		if err := tx.Commit(); err != nil {
			return errorutil.Wrap(err)
		}

		tx = nil

		return nil
	}

	dispatchQueuesInTransaction := func() error {
		var err error

		if tx, err = startTransactionIfNeeded(t.dbconn.RwConn, tx); err != nil {
			return errorutil.Wrap(err)
		}

		if err = dispatchAllQueues(t, t.queuesToNotify, tx); err != nil {
			return errorutil.Wrap(err)
		}

		if err = commitTransactionIfNeeded(); err != nil {
			return errorutil.Wrap(err)
		}

		return nil
	}

	ensureMessagesArePersistedAndDispatchQueues := func() error {
		if err := commitTransactionIfNeeded(); err != nil {
			return errorutil.Wrap(err)
		}

		if err := dispatchQueuesInTransaction(); err != nil {
			return errorutil.Wrap(err)
		}

		return nil
	}

	messagesTicker := time.NewTicker(1 * time.Second)

	var err error

	for {
		select {
		case <-messagesTicker.C:
			if err := ensureMessagesArePersistedAndDispatchQueues(); err != nil {
				return nil
			}
		case actionTuple, ok := <-t.actions:
			if !ok {
				// cancel() has been called
				// dispatch any remaning message and leave
				if err = ensureMessagesArePersistedAndDispatchQueues(); err != nil {
					return errorutil.Wrap(err)
				}

				return nil
			}

			if tx, err = executeActionInTransaction(t.dbconn.RwConn, tx, t, actionTuple); err != nil {
				return errorutil.Wrap(err)
			}
		}
	}
}

type Result [lastKey]interface{}

type ResultPublisher interface {
	Publish(Result)
}
