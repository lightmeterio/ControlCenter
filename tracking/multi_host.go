// SPDX-FileCopyrightText: 2022 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package tracking

import (
	"database/sql"
	"errors"
	"time"

	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/pkg/postfix"
	parser "gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
)

func mailSentActionCanGenerateDeliveryResult(tx *sql.Tx, r postfix.Record, trackerStmts dbconn.TxPreparedStmts, p parser.SmtpSentStatus) (bool, error) {
	// check if we are an internal relay, and it's possible that the logs from the previous hop are not yet known.
	// NOTE: For now we support only outbound relayed messages for now.
	direction := findMsgDirection(r.Header)

	if direction == MessageDirectionIncoming {
		// NOTE: we don't support relayed inbound messages yet
		return true, nil
	}

	// goes in the queue chain until finding a queue that had an authenticated connection

	var queueId int64

	queueId, err := findQueueIdFromQueueValue(p.Queue, trackerStmts)
	if err != nil {
		return false, errorutil.Wrap(err)
	}

	for {
		authSuccessCount, err := func(queueId int64) (int64, error) {
			var authSuccessCount int64

			//nolint:sqlclosecheck
			err := trackerStmts.Get(selectConnectionAuthCountForQueue).QueryRow(queueId, ConnectionAuthSuccessCount).Scan(&authSuccessCount)

			if err != nil && errors.Is(err, sql.ErrNoRows) {
				// TODO: this usually happens because the `disconnect from` happens AFTER the `mail sent=...` action
				// meaning that the SMTP connection lasted a bit longer and its end was logged a bit later.
				// this is a bit difficult to fix, as it'd force us to "schedule" the generation of a delivery attempt result.
				// FIXME: for now the workaround is just to mock the behaviour, which will result into imprecise data!
				// return false, errorutil.Wrap(err)
				return 0, nil
			}

			if err != nil {
				return 0, errorutil.Wrap(err)
			}

			return authSuccessCount, nil
		}(queueId)

		if err != nil {
			return false, errorutil.Wrap(err)
		}

		// authenticated queue, good to go!
		if authSuccessCount > 0 {
			return true, nil
		}

		var (
			parentQueueId  int64
			parentRecordId int64
		)

		// this queue was created from a connection without authentication.
		// That means we are now in a relay, where the message started elsewhere.
		err = trackerStmts.Get(selectQueueFromParentingNewQueue).QueryRow(queueId).Scan(&parentRecordId, &parentQueueId)

		if err != nil && errors.Is(err, sql.ErrNoRows) {
			// a gap was found, and it's marked to be "filled out" later
			return false, nil
		}

		if err != nil {
			return false, errorutil.Wrap(err)
		}

		// a parent was found. Search deeper...
		queueId = parentQueueId

		continue
	}
}

func handleMailSentToExternalRelay(tx *sql.Tx, r postfix.Record, trackerStmts dbconn.TxPreparedStmts, p parser.SmtpSentStatus) error {
	canDeliver, err := mailSentActionCanGenerateDeliveryResult(tx, r, trackerStmts, p)
	if err != nil {
		return errorutil.Wrap(err)
	}

	if !canDeliver {
		// there are some gaps in the logs, so we're unable to notify a new delivery now.
		// But we need to create a result and "schedule" it to be executed later,
		// once the gap is filled.
		resultInfo, err := createResult(trackerStmts, r)
		if err != nil {
			return errorutil.Wrap(err)
		}

		queueId, err := findQueueIdFromQueueValue(p.Queue, trackerStmts)
		if err != nil {
			return errorutil.Wrap(err)
		}

		// Mark result to be notified only when all the gaps in the logs are filled.
		if _, err := trackerStmts.Get(insertPreNotificationByQueueIdAndResultId).Exec(queueId, resultInfo.id); err != nil {
			return errorutil.Wrap(err)
		}

		return nil
	}

	// here we know all is good for notifying a new delivery

	// not internally queued
	err = createMailDeliveredResult(r, trackerStmts)

	if err != nil && errors.Is(err, sql.ErrNoRows) {
		// sometimes a queue has been created in a point earlier than what's known by us. Just ignore it then
		// TODO: postfix can have very long living queues (that are active for many days)
		// and can use such queue for delivering many e-mails.
		// Right now we are not notifying any of those intermediate e-mails, which might not be desisable
		// TODO: More investigation is needed
		return nil
	}

	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

const unknownConnectionId = -99

func findNewQueueIdOrCreateIncompleteOne(queue string, r postfix.Record, trackerStmts dbconn.TxPreparedStmts) (int64, error) {
	newQueueId, err := findQueueIdFromQueueValue(queue, trackerStmts)
	if err == nil {
		return newQueueId, nil
	}

	if !errors.Is(err, sql.ErrNoRows) {
		return 0, errorutil.Wrap(err)
	}

	// The queue will be created in the future, as soon as the relevant logs arrive.
	// As we don't know its connection yet, just fake it and replace it once we know it.
	queueId, err := createQueue(r.Time, unknownConnectionId, queue, r.Location, trackerStmts)
	if err != nil {
		return 0, errorutil.Wrap(err)
	}

	return queueId, nil
}

func createOrFixQueue(time time.Time, connectionId int64, queue string, loc postfix.RecordLocation, trackerStmts dbconn.TxPreparedStmts) (int64, error) {
	// If the queue already exists and is in a "incomplete" state, we have to "fix it".
	queueId, err := findQueueIdFromQueueValue(queue, trackerStmts)

	// brand new queue
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		return createQueue(time, connectionId, queue, loc, trackerStmts)
	}

	if err != nil {
		return 0, errorutil.Wrap(err)
	}

	// fix the queue, assigning the correction connection to it
	if _, err := trackerStmts.Get(fixQueueConnectionId).Exec(connectionId, queueId); err != nil {
		return 0, errorutil.Wrap(err)
	}

	if err := incrementConnectionUsage(trackerStmts, connectionId); err != nil {
		return 0, errorutil.Wrap(err)
	}

	return queueId, nil
}

type MultiNodeTypeHandler struct {
}

func (h MultiNodeTypeHandler) CreateQueue(t time.Time, connectionId int64, queue string, location postfix.RecordLocation, trackerStmts dbconn.TxPreparedStmts) (int64, error) {
	return createOrFixQueue(t, connectionId, queue, location, trackerStmts)
}

func (h *MultiNodeTypeHandler) FindQueue(queue string, r postfix.Record, stmts dbconn.TxPreparedStmts) (int64, error) {
	return findNewQueueIdOrCreateIncompleteOne(queue, r, stmts)
}

func (h *MultiNodeTypeHandler) HandleMailSentAction(tx *sql.Tx, r postfix.Record, p parser.SmtpSentStatus, trackerStmts dbconn.TxPreparedStmts) error {
	e, cast := p.ExtraMessagePayload.(parser.SmtpSentStatusExtraMessageSentQueued)

	sentToNextRelayHop := !cast || (cast && !e.InternalMTA && p.RelayPort == 25)

	// delivery to the next relay outside of the system
	if sentToNextRelayHop {
		if err := handleMailSentToExternalRelay(tx, r, trackerStmts, p); err != nil {
			return errorutil.Wrap(err)
		}

		return nil
	}

	origQueueId, err := findQueueIdFromQueueValue(p.Queue, trackerStmts)

	// TODO: this block is copy&pasted many times! It should be refactored!
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		// sometimes a queue has been created in a point earlier than what's known by us. Just ignore it then
		// TODO: postfix can have very long living queues (that are active for many days)
		// and can use such queue for delivering many e-mails.
		// Right now we are not notifying any of those intermediate e-mails, which might not be desisable
		// More investigation is needed
		return nil
	}

	if err != nil {
		return errorutil.Wrap(err)
	}

	// here we know that the current queue is not the final one, so another one will be used to deliver
	// to the final destination
	newQueueId, err := findNewQueueIdOrCreateIncompleteOne(e.Queue, r, trackerStmts)
	if err != nil {
		return errorutil.Wrap(err)
	}

	// this is an e-mail that postfix sometimes (if configured to do so) sends to itself before trying to deliver.
	// As it's moved to another queue to be delivered, we queue the original and the newly created queue
	//nolint:sqlclosecheck
	_, err = trackerStmts.Get(insertQueueParenting).Exec(origQueueId, newQueueId, queueParentingRelayType)
	if err != nil {
		return errorutil.Wrap(err)
	}

	// by this point we might have already "filled the gap" of a replayed message that was waiting for its
	// first "hop" to be filled out.
	// if so, we dispatch it, cleaning the whole "chain".
	var resultId int64

	err = trackerStmts.Get(selectPreNotificationResultIdsForQueue).QueryRow(newQueueId).Scan(&resultId)

	if err != nil && errors.Is(err, sql.ErrNoRows) {
		// No result in prenotification state yet. All good, life continues...
		return nil
	}

	if err != nil {
		return errorutil.Wrap(err)
	}

	if _, err := trackerStmts.Get(deletePreNotificationEntryByQueueId).Exec(newQueueId); err != nil {
		return errorutil.Wrap(err)
	}

	if err := markResultToBeNotified(trackerStmts, resultId); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}
