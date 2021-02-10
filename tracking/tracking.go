package tracking

import (
	"database/sql"
	"errors"
	"github.com/rs/zerolog/log"
	"gitlab.com/lightmeter/controlcenter/data"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/migrator"
	parser "gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser"
	"gitlab.com/lightmeter/controlcenter/pkg/runner"
	_ "gitlab.com/lightmeter/controlcenter/tracking/migrations"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"path"
	"reflect"
	"time"
)

type MessageDirection int

const (
	// NOTE: those values are stored in the database,
	// so changing them must force a data migration to new values!
	MessageDirectionOutbound MessageDirection = 0
	MessageDirectionIncoming MessageDirection = 1
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
	PickupActionType
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
	PickupActionType:            {impl: pickupAction},
}

type trackerStmts [lastTrackerStmtKey]*sql.Stmt

type Tracker struct {
	runner.CancelableRunner

	stmts                trackerStmts
	dbconn               dbconn.ConnPair
	actions              chan actionTuple
	txActions            <-chan func(*sql.Tx) error
	resultsToNotify      chan resultInfo
	queuesCommitNotifier *queuesCommitNotifier
}

func (t *Tracker) MostRecentLogTime() time.Time {
	queryConnection := `select value from connection_data where key in (?,?) order by id desc limit 1`
	queryResult := `select value from result_data where key = ? order by id desc limit 1`
	queryQueue := `select value from queue_data where key in (?,?) order by id desc limit 1`

	exec := func(query string, args ...interface{}) int64 {
		var ts int64
		err := t.dbconn.RoConn.QueryRow(query, args...).Scan(&ts)

		if errors.Is(err, sql.ErrNoRows) {
			return 0
		}

		errorutil.MustSucceed(err)

		return ts
	}

	v := int64(0)

	for _, p := range []struct {
		q    string
		args []interface{}
	}{
		{q: queryConnection, args: []interface{}{ConnectionBeginKey, ConnectionEndKey}},
		{q: queryResult, args: []interface{}{ResultDeliveryTimeKey}},
		{q: queryQueue, args: []interface{}{QueueBeginKey, QueueEndKey}},
	} {
		r := exec(p.q, p.args...)
		if r > v {
			v = r
		}
	}

	if v == 0 {
		return time.Time{}
	}

	return time.Unix(v, 0).In(time.UTC)
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

	txActions := make(chan func(*sql.Tx) error, 1024*1000)

	resultsToNotify := make(chan resultInfo, 1024*1000)

	queuesCommitNotifier := &queuesCommitNotifier{
		resultsToNotify: resultsToNotify,
		publisher:       pub,
	}

	trackerActions := make(chan actionTuple, 1024*1000)

	tracker := &Tracker{
		stmts:                trackerStmts,
		dbconn:               conn,
		actions:              trackerActions,
		txActions:            txActions,
		resultsToNotify:      resultsToNotify,
		queuesCommitNotifier: queuesCommitNotifier,
	}

	queuesCommitNotifier.CancelableRunner = runner.NewCancelableRunner(func(done runner.DoneChan, _ runner.CancelChan) {
		go func() {
			done <- func() error {
				// will leave when resultsToNotify is closed
				if err := runResultsNotifier(notifierStmts, queuesCommitNotifier, trackerStmts, txActions); err != nil {
					return errorutil.Wrap(err)
				}

				close(txActions)

				return nil
			}()
		}()
	})

	// TODO: cleanup this cancel/waitForDone code that is a total mess and impossible to understand!!!
	tracker.CancelableRunner = runner.NewCancelableRunner(func(done runner.DoneChan, cancel runner.CancelChan) {
		notifierDone, _ := queuesCommitNotifier.Run()

		go func() {
			<-cancel

			close(tracker.actions)
		}()

		go func() {
			runTrackerChan := make(chan error)
			waitNotifierChan := make(chan error)

			go func() {
				runTrackerChan <- func() error {
					if err := runTracker(tracker); err != nil {
						return errorutil.Wrap(err)
					}

					return nil
				}()

				runTrackerChan <- err
			}()

			go func() {
				waitNotifierChan <- notifierDone()
			}()

			var err error

			select {
			case e := <-runTrackerChan:
				err = e
				errorutil.MustSucceed(<-waitNotifierChan)
			case e := <-waitNotifierChan:
				err = e
				errorutil.MustSucceed(<-runTrackerChan)
			}

			errorutil.MustSucceed(err)

			done <- err
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
		log.Panic().Msgf("SPANK SPANK: Invalid/unsupported action!: %v", actionTuple)
	}

	action := actionRecord.impl
	actionDataPair := actionTuple.actionDataPair

	if tx, err = startTransactionIfNeeded(conn, tx); err != nil {
		return nil, errorutil.Wrap(err)
	}

	if err = action(t, tx, actionTuple.record, actionDataPair); err != nil {
		if err, isDeletionError := errorutil.ErrorAs(err, &DeletionError{}); isDeletionError {
			//nolint:errorlint
			asDeletionError := err.(*DeletionError)
			// FIXME: For now we are ignoring some errors that happen during deletion of unused queues
			// but we should investigate and make and fix them!
			// As a result, we are keeping old data, that failed to be deleted, to accumulate in the database
			// and potentially making queries slower... :-(
			log.Warn().Msgf("--------- Ignoring error deleting data triggered by log on file: %v:%v, message: %v",
				asDeletionError.Loc.Filename, asDeletionError.Loc.Line, asDeletionError)

			return tx, nil
		}

		return nil, errorutil.Wrap(err, "log file: ", actionTuple.record.Location.Filename, ":", actionTuple.record.Location.Line)
	}

	return tx, nil
}

func dispatchQueuesInTransaction(tx *sql.Tx, t *Tracker) (*sql.Tx, error) {
	var err error

	if tx, err = startTransactionIfNeeded(t.dbconn.RwConn, tx); err != nil {
		return nil, errorutil.Wrap(err)
	}

	if err = dispatchAllResults(t, t.resultsToNotify, tx); err != nil {
		return nil, errorutil.Wrap(err)
	}

	if err = commitTransactionIfNeeded(tx); err != nil {
		return nil, errorutil.Wrap(err)
	}

	tx = nil

	return tx, nil
}

func commitTransactionIfNeeded(tx *sql.Tx) error {
	if tx == nil {
		return nil
	}

	if err := tx.Commit(); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func ensureMessagesArePersistedAndDispatchResults(tx *sql.Tx, t *Tracker) (*sql.Tx, error) {
	var err error

	if err = commitTransactionIfNeeded(tx); err != nil {
		return nil, errorutil.Wrap(err)
	}

	tx = nil

	if tx, err = dispatchQueuesInTransaction(tx, t); err != nil {
		return nil, errorutil.Wrap(err)
	}

	return tx, nil
}

func handleTxAction(tx *sql.Tx, t *Tracker, ok bool, recv reflect.Value) (*sql.Tx, bool, error) {
	var err error

	if !ok {
		if err = commitTransactionIfNeeded(tx); err != nil {
			return nil, false, errorutil.Wrap(err)
		}

		return tx, false, nil
	}

	if tx, err = startTransactionIfNeeded(t.dbconn.RwConn, tx); err != nil {
		return nil, false, errorutil.Wrap(err)
	}

	txAction := recv.Interface().(func(*sql.Tx) error)

	err = txAction(tx)

	if err != nil {
		if err, isDeletionError := errorutil.ErrorAs(err, &DeletionError{}); isDeletionError {
			//nolint:errorlint
			asDeletionError := err.(*DeletionError)
			// FIXME: For now we are ignoring some errors that happen during deletion of unused queues
			// but we should investigate and make and fix them!
			// As a result, we are keeping old data, that failed to be deleted, to accumulate in the database
			// and potentially making queries slower... :-(
			log.Warn().Msgf("--------- Ignoring error deleting data triggered by log on file: %v:%v, message: %v",
				asDeletionError.Loc.Filename, asDeletionError.Loc.Line, asDeletionError)

			return tx, true, nil
		}

		errorutil.MustSucceed(err)

		return nil, false, errorutil.Wrap(err)
	}

	return tx, true, nil
}

func runTracker(t *Tracker) error {
	var (
		tx  *sql.Tx
		err error
	)

	messagesTicker := time.NewTicker(500 * time.Millisecond)

	txActionsAsValue := reflect.ValueOf(t.txActions)
	tickerAsValue := reflect.ValueOf(messagesTicker.C)
	messageActionsAsValue := reflect.ValueOf(t.actions)

	branches := []reflect.SelectCase{
		{Dir: reflect.SelectRecv, Chan: txActionsAsValue},
		{Dir: reflect.SelectRecv, Chan: tickerAsValue},
		{Dir: reflect.SelectRecv, Chan: messageActionsAsValue},
	}

loop:
	for {
		chosen, recv, ok := reflect.Select(branches)

		switch chosen {
		case 0:
			var shouldContinue bool

			tx, shouldContinue, err = handleTxAction(tx, t, ok, recv)
			if err != nil {
				return errorutil.Wrap(err)
			}

			if !shouldContinue {
				break loop
			}
		case 1:
			if tx, err = ensureMessagesArePersistedAndDispatchResults(tx, t); err != nil {
				return errorutil.Wrap(err)
			}
		case 2:
			if !ok {
				// cancel() has been called
				if tx, err = ensureMessagesArePersistedAndDispatchResults(tx, t); err != nil {
					return errorutil.Wrap(err)
				}

				close(t.resultsToNotify)

				// Remove this branch from the select, as no new messages should arrive
				// from now on
				branches = branches[0 : len(branches)-1]

				break
			}

			actionTuple := recv.Interface().(actionTuple)

			if tx, err = executeActionInTransaction(t.dbconn.RwConn, tx, t, actionTuple); err != nil {
				errorutil.MustSucceed(err)
				return errorutil.Wrap(err)
			}
		default:
			panic("Read wrong select index!!!")
		}
	}

	return nil
}

type Result [lastKey]interface{}

type ResultPublisher interface {
	Publish(Result)
}
