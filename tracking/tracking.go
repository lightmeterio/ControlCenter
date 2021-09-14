// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package tracking

import (
	"database/sql"
	"errors"
	"github.com/rs/zerolog/log"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/pkg/postfix"
	parser "gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser"
	"gitlab.com/lightmeter/controlcenter/pkg/runner"
	_ "gitlab.com/lightmeter/controlcenter/tracking/migrations"
	"gitlab.com/lightmeter/controlcenter/util/closeutil"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"reflect"
	"sync"
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
	MilterRejectActionType
	RejectActionType
	MessageExpiredActionType
)

type actionTuple struct {
	actionType     ActionType
	record         postfix.Record
	actionDataPair actionDataPair
}

type actionDataPair struct {
	connectionActionData connectionActionData
	resultActionData     resultActionData
}

type Publisher struct {
	actions chan<- actionTuple
}

func (p *Publisher) Publish(r postfix.Record) {
	actionType, actionDataPair := actionTypeForRecord(r)

	if actionType != UnsupportedActionType {
		p.actions <- actionTuple{
			actionType:     actionType,
			actionDataPair: actionDataPair,
			record:         r,
		}
	}
}

type actionImpl func(*Tracker, *sql.Tx, postfix.Record, actionDataPair) error

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
	MilterRejectActionType:      {impl: milterRejectAction},
	RejectActionType:            {impl: rejectAction},
	MessageExpiredActionType:    {impl: messageExpiredAction},
}

type trackerStmts = dbconn.PreparedStmts

type txActions struct {
	size    uint
	actions [resultInfosCapacity]func(*sql.Tx) error
}

type resultsNotifiers []*resultsNotifier

type Tracker struct {
	runner.CancellableRunner
	closeutil.Closers

	stmts            trackerStmts
	dbconn           *dbconn.PooledPair
	actions          chan actionTuple
	txActions        <-chan txActions
	resultsToNotify  chan resultInfos
	resultsNotifiers resultsNotifiers
}

func (t *Tracker) MostRecentLogTime() (time.Time, error) {
	conn, release := t.dbconn.RoConnPool.Acquire()

	defer release()

	queryConnection := `select value from connection_data where key in (?,?) order by rowid desc limit 1`
	queryResult := `select value from result_data where key = ? order by rowid desc limit 1`
	queryQueue := `select value from queue_data where key in (?,?) order by rowid desc limit 1`

	exec := func(query string, args ...interface{}) (int64, error) {
		var ts int64
		err := conn.QueryRow(query, args...).Scan(&ts)

		if err != nil && errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}

		if err != nil {
			return 0, errorutil.Wrap(err)
		}

		return ts, nil
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
		r, err := exec(p.q, p.args...)

		if err != nil {
			return time.Time{}, errorutil.Wrap(err)
		}

		if r > v {
			v = r
		}
	}

	if v == 0 {
		return time.Time{}, nil
	}

	return time.Unix(v, 0).In(time.UTC), nil
}

func (t *Tracker) Publisher() *Publisher {
	return &Publisher{actions: t.actions}
}

func buildResultsNotifier(
	id int,
	wg *sync.WaitGroup,
	pool *dbconn.RoPool,
	resultsToNotify <-chan resultInfos,
	pub ResultPublisher,
	trackerStmts trackerStmts,
	txActions chan txActions,
) *resultsNotifier {
	resultsNotifier := &resultsNotifier{
		resultsToNotify: resultsToNotify,
		publisher:       pub,
		id:              id,
	}

	roConn, releaseConn := pool.Acquire()

	resultsNotifier.CancellableRunner = runner.NewCancellableRunner(func(done runner.DoneChan, _ runner.CancelChan) {
		go func() {
			done <- func() error {
				log.Debug().Msgf("Tracking notifier %d has just started!", resultsNotifier.id)

				// will leave when resultsToNotify is closed
				if err := runResultsNotifier(roConn, resultsNotifier, trackerStmts, txActions); err != nil {
					return errorutil.Wrap(err)
				}

				wg.Done()

				releaseConn()

				log.Debug().Msgf("Tracking notifier %d has just ended with %v processed", resultsNotifier.id, resultsNotifier.counter)

				return nil
			}()
		}()
	})

	return resultsNotifier
}

const numberOfNotifiers = 1

func New(conn *dbconn.PooledPair, pub ResultPublisher) (*Tracker, error) {
	trackerStmts := make(dbconn.PreparedStmts, lastTrackerStmtKey)
	if err := dbconn.PrepareRwStmts(trackerStmtsText, conn.RwConn, trackerStmts); err != nil {
		return nil, errorutil.Wrap(err)
	}

	if err := conn.RoConnPool.ForEach(prepareCommitterConnection); err != nil {
		return nil, errorutil.Wrap(err)
	}

	txActions := make(chan txActions, 1024*10)
	resultsToNotify := make(chan resultInfos, 1024*10)
	trackerActions := make(chan actionTuple, 1024*1000)

	wg := sync.WaitGroup{}

	wg.Add(numberOfNotifiers)

	tracker := &Tracker{
		stmts:           trackerStmts,
		dbconn:          conn,
		actions:         trackerActions,
		txActions:       txActions,
		resultsToNotify: resultsToNotify,
		Closers:         closeutil.New(trackerStmts),
	}

	// it should be refactored ASAP!!!!
	for i := 0; i < numberOfNotifiers; i++ {
		resultsNotifier := buildResultsNotifier(i, &wg, conn.RoConnPool, resultsToNotify, pub, trackerStmts, txActions)
		tracker.resultsNotifiers = append(tracker.resultsNotifiers, resultsNotifier)
	}

	// TODO: cleanup this cancel/waitForDone code that is a total mess and impossible to understand!!!
	tracker.CancellableRunner = runner.NewCancellableRunner(func(done runner.DoneChan, cancel runner.CancelChan) {
		go func() {
			wg.Wait()

			close(txActions)
		}()

		go func() {
			<-cancel

			close(tracker.actions)
		}()

		go func() {
			wg := sync.WaitGroup{}

			// +1 because of the tracker run
			wg.Add(numberOfNotifiers + 1)

			notifiersDones := make(chan func() error, numberOfNotifiers)

			// run tracker
			go func() {
				err := runTracker(tracker)
				errorutil.MustSucceed(err)
				wg.Done()
			}()

			// start each notifier
			go func() {
				for _, resultsNotifier := range tracker.resultsNotifiers {
					notifierDone, _ := runner.Run(resultsNotifier)
					notifiersDones <- notifierDone
				}
			}()

			// wait for notifiers to finish
			go func() {
				for done := range notifiersDones {
					err := done()
					errorutil.MustSucceed(err)
					wg.Done()
				}
			}()

			wg.Wait()

			done <- nil
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
			//nolint:errorlint,forcetypeassert
			asDeletionError := err.(*DeletionError)
			// FIXME: For now we are ignoring some errors that happen during deletion of unused queues
			// but we should investigate and make and fix them!
			// As a result, we are keeping old data, that failed to be deleted, to accumulate in the database
			// and potentially making queries slower... :-(
			log.Warn().Msgf("--------- (action) Ignoring error deleting data triggered by log on file: %v:%v, message: %v",
				asDeletionError.Loc.Filename, asDeletionError.Loc.Line, asDeletionError)

			return tx, nil
		}

		return nil, errorutil.Wrap(err, "log file: ", actionTuple.record.Location.Filename, ":", actionTuple.record.Location.Line)
	}

	return tx, nil
}

func dispatchQueuesInTransaction(tx *sql.Tx, t *Tracker, batchId int64) (*sql.Tx, error) {
	var err error

	if tx, err = startTransactionIfNeeded(t.dbconn.RwConn, tx); err != nil {
		return nil, errorutil.Wrap(err)
	}

	if err = dispatchAllResults(t, t.resultsToNotify, tx, batchId); err != nil {
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

func ensureMessagesArePersistedAndDispatchResults(tx *sql.Tx, t *Tracker, batchId int64) (*sql.Tx, error) {
	var err error

	if err = commitTransactionIfNeeded(tx); err != nil {
		return nil, errorutil.Wrap(err)
	}

	tx = nil

	if tx, err = dispatchQueuesInTransaction(tx, t, batchId); err != nil {
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

	//nolint:forcetypeassert
	txActions := recv.Interface().(txActions)

	for i := uint(0); i < txActions.size; i++ {
		txAction := txActions.actions[i]
		err = txAction(tx)

		if err != nil {
			if err, isDeletionError := errorutil.ErrorAs(err, &DeletionError{}); isDeletionError {
				//nolint:errorlint,forcetypeassert
				asDeletionError := err.(*DeletionError)
				// FIXME: For now we are ignoring some errors that happen during deletion of unused queues
				// but we should investigate and make and fix them!
				// As a result, we are keeping old data, that failed to be deleted, to accumulate in the database
				// and potentially making queries slower... :-(
				log.Warn().Msgf("--------- (txAction) Ignoring error deleting data triggered by log on file: %v:%v, message: %v",
					asDeletionError.Loc.Filename, asDeletionError.Loc.Line, asDeletionError)

				for _, v := range asDeletionError.Err.Chain().JSON() {
					log.Debug().Msgf("%v -> %v:%v", v.Error, v.File, v.Line)
				}

				return tx, true, nil
			}

			errorutil.MustSucceed(err)

			return nil, false, errorutil.Wrap(err)
		}
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

	batchId := int64(0)

loop:
	for {
		chosen, recv, ok := reflect.Select(branches)

		switch chosen {
		// actions from the notifier
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
			// ticker timeout
			if tx, err = ensureMessagesArePersistedAndDispatchResults(tx, t, batchId); err != nil {
				return errorutil.Wrap(err)
			}
			batchId++
		case 2:
			// new action from the logs
			if !ok {
				// cancel() has been called
				if tx, err = ensureMessagesArePersistedAndDispatchResults(tx, t, batchId); err != nil {
					return errorutil.Wrap(err)
				}

				close(t.resultsToNotify)

				// Remove this branch from the select, as no new messages should arrive
				// from now on
				branches = branches[0 : len(branches)-1]

				break
			}

			//nolint:forcetypeassert
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
