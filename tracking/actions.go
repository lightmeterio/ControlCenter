// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package tracking

import (
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/pkg/postfix"
	parser "gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
)

type MessageDirection int

const (
	// NOTE: those values are stored in the database,
	// so changing them must force a data migration to new values!
	MessageDirectionOutbound MessageDirection = 0
	MessageDirectionIncoming MessageDirection = 1
)

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
	LightmeterHeaderDumpActionType
	LightmeterRelayedBounceType
)

var actions = map[ActionType]actionRecord{
	ConnectActionType:              {impl: connectAction},
	CloneActionType:                {impl: cloneAction},
	CleanupProcessingActionType:    {impl: cleanupProcessingAction},
	MailQueuedActionType:           {impl: mailQueuedAction},
	DisconnectActionType:           {impl: disconnectAction},
	MailSentActionType:             {impl: mailSentAction},
	CommitActionType:               {impl: commitAction},
	MailBouncedActionType:          {impl: mailBouncedAction},
	BounceCreatedActionType:        {impl: bounceCreatedAction},
	PickupActionType:               {impl: pickupAction},
	MilterRejectActionType:         {impl: milterRejectAction},
	RejectActionType:               {impl: rejectAction},
	MessageExpiredActionType:       {impl: messageExpiredAction},
	LightmeterHeaderDumpActionType: {impl: lightmeterHeaderDumpAction},
	LightmeterRelayedBounceType:    {impl: lightmeterRelayedBounceAction},
}

var emptyActionDataPair = actionDataPair{connectionActionData: nil, resultActionData: nil}

func actionTypeForRecord(r postfix.Record) (ActionType, actionDataPair) {
	// FIXME: this function will get uglier over time :-(
	switch p := r.Payload.(type) {
	case parser.SmtpSentStatus:
		switch p.Status {
		case parser.BouncedStatus:
			return MailBouncedActionType, emptyActionDataPair
		case parser.SentStatus, parser.ReceivedStatus:
			return MailSentActionType, emptyActionDataPair
		case parser.DeferredStatus:
			return MailBouncedActionType, emptyActionDataPair
		case parser.ExpiredStatus:
			fallthrough
		case parser.ReturnedStatus:
			fallthrough
		case parser.RepliedStatus:
			fallthrough
		default:
			return UnsupportedActionType, emptyActionDataPair
		}
	case parser.SmtpdConnect:
		return ConnectActionType, emptyActionDataPair
	case parser.SmtpdDisconnect:
		return DisconnectActionType, emptyActionDataPair
	case parser.SmtpdMailAccepted:
		return CloneActionType, emptyActionDataPair
	case parser.BounceCreated:
		return BounceCreatedActionType, emptyActionDataPair
	case parser.QmgrMailQueued:
		return MailQueuedActionType, emptyActionDataPair
	case parser.CleanupMessageAccepted:
		return CleanupProcessingActionType, emptyActionDataPair
	case parser.QmgrRemoved:
		return CommitActionType, emptyActionDataPair
	case parser.Pickup:
		return PickupActionType, emptyActionDataPair
	case parser.CleanupMilterReject:
		return MilterRejectActionType, emptyActionDataPair
	case parser.SmtpdReject:
		return RejectActionType, emptyActionDataPair
	case parser.QmgrMessageExpired:
		return MessageExpiredActionType, emptyActionDataPair
	case parser.LightmeterDumpedHeader:
		return LightmeterHeaderDumpActionType, emptyActionDataPair
	case parser.LightmeterRelayedBounce:
		return LightmeterRelayedBounceType, emptyActionDataPair
	}

	return UnsupportedActionType, emptyActionDataPair
}

func insertConnectionWithPid(trackerStmts dbconn.TxPreparedStmts, pidId int64) (int64, error) {
	//nolint:sqlclosecheck
	result, err := trackerStmts.Get(insertConnectionOnConnection).Exec(pidId)
	if err != nil {
		return 0, errorutil.Wrap(err)
	}

	connectionId, err := result.LastInsertId()
	if err != nil {
		return 0, errorutil.Wrap(err)
	}

	return connectionId, nil
}

func insertPid(trackerStmts dbconn.TxPreparedStmts, pid int, host string) (int64, error) {
	// TODO: check if there's already a connection there, as it should not be
	// in case there be, it means some message has been lost in the way
	//nolint:sqlclosecheck
	result, err := trackerStmts.Get(insertPidOnConnection).Exec(pid, host)
	if err != nil {
		return 0, errorutil.Wrap(err)
	}

	pidId, err := result.LastInsertId()
	if err != nil {
		return 0, errorutil.Wrap(err)
	}

	return pidId, nil
}

func acquirePid(trackerStmts dbconn.TxPreparedStmts, pid int, host string) (int64, error) {
	var pidId int64

	//nolint:sqlclosecheck
	err := trackerStmts.Get(selectPidForPidAndHost).QueryRow(pid, host).Scan(&pidId)
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		// Create new pid
		pidId, err := insertPid(trackerStmts, pid, host)

		if err != nil {
			return 0, errorutil.Wrap(err)
		}

		return pidId, nil
	}

	if err != nil {
		return 0, errorutil.Wrap(err)
	}

	err = incrementPidUsage(trackerStmts, pidId)
	if err != nil {
		return 0, errorutil.Wrap(err)
	}

	// reuse existing pid
	return pidId, nil
}

func createConnection(r postfix.Record, trackerStmts dbconn.TxPreparedStmts) (int64, error) {
	pidId, err := acquirePid(trackerStmts, r.Header.PID, r.Header.Host)
	if err != nil {
		return 0, errorutil.Wrap(err)
	}

	connectionId, err := insertConnectionWithPid(trackerStmts, pidId)
	if err != nil {
		return 0, errorutil.Wrap(err)
	}

	return connectionId, nil
}

func connectAction(tx *sql.Tx, r postfix.Record, actionDataPair actionDataPair, trackerStmts dbconn.TxPreparedStmts) error {
	// TODO: check if there's already a connection there, as it should not be
	// in case there be, it means some message has been lost in the way
	connectionId, err := createConnection(r, trackerStmts)
	if err != nil {
		return errorutil.Wrap(err)
	}

	// TODO: there might be other payloads about connection, so this cast is not always safe
	//nolint:forcetypeassert
	payload := r.Payload.(parser.SmtpdConnect)

	//nolint:sqlclosecheck
	_, err = trackerStmts.Get(insertConnectionDataFourRows).Exec(
		connectionId, ConnectionBeginKey, r.Time.Unix(),
		connectionId, ConnectionClientHostnameKey, payload.Host,
		connectionId, ConnectionFilenameKey, r.Location.Filename,
		connectionId, ConnectionLineKey, r.Location.Line,
	)

	if err != nil {
		return errorutil.Wrap(err)
	}

	// no IP (usually postfix sees it as "unknown"), just ignore it
	// TODO: this should be a supported use case, but I don't know what to do in this case!
	if payload.IP == nil {
		log.Warn().Msgf("Ignoring unknown IP on connection on file %v:%v", r.Location.Filename, r.Location.Line)
		return nil
	}

	//nolint:sqlclosecheck
	_, err = trackerStmts.Get(insertConnectionData).Exec(connectionId, ConnectionClientIPKey, payload.IP)
	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func findConnectionIdAndUsageCounter(trackerStmts dbconn.TxPreparedStmts, h parser.Header) (int64, int, error) {
	var (
		connectionId int64
		usageCounter int
	)

	// find a connection entry for this
	//nolint:sqlclosecheck
	err := trackerStmts.Get(selectConnectionAndUsageCounterForPid).QueryRow(h.Host, h.PID).Scan(&connectionId, &usageCounter)

	if err != nil {
		return 0, 0, errorutil.Wrap(err)
	}

	return connectionId, usageCounter, nil
}

type kvData struct {
	key   uint
	value interface{}
}

func insertQueueDataValues(stmts dbconn.TxPreparedStmts, queueId int64, values ...kvData) error {
	for _, v := range values {
		//nolint:sqlclosecheck
		if _, err := stmts.Get(insertQueueData).Exec(queueId, v.key, v.value); err != nil {
			return errorutil.Wrap(err)
		}
	}

	return nil
}

func insertResultDataValues(stmts dbconn.TxPreparedStmts, resultId int64, values ...kvData) error {
	for _, v := range values {
		//nolint:sqlclosecheck
		if _, err := stmts.Get(insertResultData).Exec(resultId, v.key, v.value); err != nil {
			return errorutil.Wrap(err)
		}
	}

	return nil
}

func createQueue(time time.Time, connectionId int64, queue string, loc postfix.RecordLocation, trackerStmts dbconn.TxPreparedStmts) (int64, error) {
	//nolint:sqlclosecheck
	result, err := trackerStmts.Get(insertQueueForConnection).Exec(connectionId, queue)
	if err != nil {
		return 0, errorutil.Wrap(err)
	}

	queueId, err := result.LastInsertId()
	if err != nil {
		return 0, errorutil.Wrap(err)
	}

	err = insertQueueDataValues(trackerStmts, queueId,
		kvData{key: QueueBeginKey, value: time.Unix()},
		kvData{key: QueueFilenameKey, value: loc.Filename},
		kvData{key: QueueLineKey, value: loc.Line},
	)

	if err != nil {
		return 0, errorutil.Wrap(err)
	}

	err = incrementConnectionUsage(trackerStmts, connectionId)
	if err != nil {
		return 0, errorutil.Wrap(err)
	}

	return queueId, nil
}

// assign a queue, just created.
// find the connection with a given pid, and append the queue to the connection
func cloneAction(tx *sql.Tx, r postfix.Record, actionDataPair actionDataPair, trackerStmts dbconn.TxPreparedStmts) error {
	//nolint:forcetypeassert
	p := r.Payload.(parser.SmtpdMailAccepted)

	connectionId, _, err := findConnectionIdAndUsageCounter(trackerStmts, r.Header)
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		log.Warn().Msgf("Connection for line %v not found", r.Location)
		return nil
	}

	if err != nil {
		return errorutil.Wrap(err)
	}

	if _, err := createOrFixQueue(r.Time, connectionId, p.Queue, r.Location, trackerStmts); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func incrementConnectionUsage(stmts dbconn.TxPreparedStmts, connectionId int64) error {
	//nolint:sqlclosecheck
	_, err := stmts.Get(incrementConnectionUsageById).Exec(connectionId)
	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func decrementConnectionUsage(stmts dbconn.TxPreparedStmts, connectionId int64) error {
	//nolint:sqlclosecheck
	_, err := stmts.Get(decrementConnectionUsageById).Exec(connectionId)
	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

// associate a queue to a message-id
func cleanupProcessingAction(tx *sql.Tx, r postfix.Record, actionDataPair actionDataPair, trackerStmts dbconn.TxPreparedStmts) error {
	//nolint:forcetypeassert
	p := r.Payload.(parser.CleanupMessageAccepted)

	queueId, err := func() (int64, error) {
		queueId, err := findQueueIdFromQueueValue(p.Queue, trackerStmts)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return 0, errorutil.Wrap(err)
		}

		if err == nil {
			return queueId, nil
		}

		// Create a dummy connection with no data, meaning it's been generated by the server itself, not via SMTP
		connectionId, err := createConnection(r, trackerStmts)
		if err != nil {
			return 0, errorutil.Wrap(err)
		}

		// Then a queue for it

		queueId, err = createQueue(r.Time, connectionId, p.Queue, r.Location, trackerStmts)
		if err != nil {
			return 0, errorutil.Wrap(err)
		}

		return queueId, nil
	}()

	if err != nil {
		return errorutil.Wrap(err)
	}

	err = insertQueueDataValues(trackerStmts, queueId,
		kvData{key: QueueMessageIDKey, value: p.MessageId},
		kvData{key: MessageIdFilenameKey, value: r.Location.Filename},
		kvData{key: MessageIdLineKey, value: r.Location.Line},
		kvData{key: MessageIdIsCorruptedKey, value: p.Corrupted},
	)

	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func findQueueIdFromQueueValue(queue string, trackerStmts dbconn.TxPreparedStmts) (int64, error) {
	var queueId int64

	//nolint:sqlclosecheck
	err := trackerStmts.Get(selectQueueIdForQueue).QueryRow(queue).Scan(&queueId)

	if err != nil {
		return 0, errorutil.Wrap(err, "No queue id for queue: ", queue)
	}

	return queueId, nil
}

func mailQueuedAction(tx *sql.Tx, r postfix.Record, actionDataPair actionDataPair, trackerStmts dbconn.TxPreparedStmts) error {
	// I have the queue id and need to set the e-mail sender, size and nrcpt
	//nolint:forcetypeassert
	p := r.Payload.(parser.QmgrMailQueued)

	queueId, err := findNewQueueIdOrCreateIncompleteOne(p.Queue, r, trackerStmts)
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		// started reading the logs when a queue is referenced, but not known (it was on a previous and unknown log)
		// just ignore it.
		return nil
	}

	if err != nil {
		return errorutil.Wrap(err)
	}

	err = insertQueueDataValues(trackerStmts, queueId,
		kvData{key: QueueSenderLocalPartKey, value: p.SenderLocalPart},
		kvData{key: QueueSenderDomainPartKey, value: p.SenderDomainPart},
		kvData{key: QueueOriginalMessageSizeKey, value: p.Size},
		kvData{key: QueueNRCPTKey, value: p.Nrcpt},
	)

	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func disconnectAction(tx *sql.Tx, r postfix.Record, actionDataPair actionDataPair, trackerStmts dbconn.TxPreparedStmts) error {
	connectionId, usageCounter, err := findConnectionIdAndUsageCounter(trackerStmts, r.Header)

	// it's possible for a "disconnect" not to have a "connect", if I started reading the log
	// in between the two lines. In such cases, I just ignore the line.
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		log.Warn().Msgf("Could not find a connection in log file: %v:%v", r.Location.Filename, r.Location.Line)
		return nil
	}

	if err != nil {
		return errorutil.Wrap(err)
	}

	//nolint:sqlclosecheck
	_, err = trackerStmts.Get(insertConnectionData).Exec(connectionId, ConnectionEndKey, r.Time.Unix())
	if err != nil {
		return errorutil.Wrap(err)
	}

	authSuccess := func() int {
		//nolint:forcetypeassert
		disconnect := r.Payload.(parser.SmtpdDisconnect)

		if auth, ok := disconnect.Stats["auth"]; ok {
			return auth.Success
		}

		return 0
	}()

	//nolint:sqlclosecheck
	_, err = trackerStmts.Get(insertConnectionData).Exec(connectionId, ConnectionAuthSuccessCount, authSuccess)
	if err != nil {
		return errorutil.Wrap(err)
	}

	if usageCounter > 0 {
		// Cannot delete the connection yet as there are queues using it!
		return nil
	}

	err = deleteConnection(trackerStmts, connectionId)
	if err != nil {
		return errorutil.Wrap(err)
	}

	// TODO: consider checking if the connection data matches
	// client hostname and ip are the same

	return nil
}

type queueParentingType int

const (
	// Beware that whose values are stored in the database, so please very careful on changing them!
	// As a general rule, you can add new values in the end of the list, but not change their value or meaning
	queueParentingRelayType          = 0
	queueParentingBounceCreationType = 1
)

func createMailDeliveredResult(r postfix.Record, trackerStmts dbconn.TxPreparedStmts) error {
	resultInfo, err := createResult(trackerStmts, r)
	if err != nil {
		return errorutil.Wrap(err)
	}

	err = markResultToBeNotified(trackerStmts, resultInfo.id)
	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

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
				// TODO: this usually happens because the `disconnnect from` happens AFTER the `mail sent=...` action
				// meaning that the SMTP connection lasted a bit longer and its end was logged a bit later.
				// this is a bit difficult to fix, as it'd force us to "schedule" the generation of a delivery attempt result.
				// FIXME: for now the workaround is just to mock the behaviour, which will result into imprecise data!
				//return false, errorutil.Wrap(err)
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

func mailSentAction(tx *sql.Tx, r postfix.Record, actionDataPair actionDataPair, trackerStmts dbconn.TxPreparedStmts) error {
	// Check if message has been forwarded to the an internal relay
	//nolint:forcetypeassert
	p := r.Payload.(parser.SmtpSentStatus)

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

func markResultToBeNotified(trackerStmts dbconn.TxPreparedStmts, resultId int64) error {
	//nolint:sqlclosecheck
	_, err := trackerStmts.Get(insertNotificationQueue).Exec(resultId)
	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func commitAction(tx *sql.Tx, r postfix.Record, actionDataPair actionDataPair, trackerStmts dbconn.TxPreparedStmts) error {
	//nolint:forcetypeassert
	p := r.Payload.(parser.QmgrRemoved)

	queueId, err := findQueueIdFromQueueValue(p.Queue, trackerStmts)

	if err != nil && errors.Is(err, sql.ErrNoRows) {
		log.Warn().Msgf("Could not find queue %v for outbound e-mail, therefore ignoring it! On %v:%v", p.Queue, r.Location.Filename, r.Location.Line)
		return nil
	}

	if err != nil {
		return errorutil.Wrap(err)
	}

	err = insertQueueDataValues(trackerStmts, queueId,
		kvData{key: QueueEndKey, value: r.Time.Unix()},
		kvData{key: QueueCommitFilenameKey, value: r.Location.Filename},
		kvData{key: QueueCommitLineKey, value: r.Location.Line},
	)

	if err != nil {
		return errorutil.Wrap(err)
	}

	_, err = tryToDeleteQueue(trackerStmts, queueId, r.Location)
	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func findMsgDirection(h parser.Header) MessageDirection {
	if strings.HasSuffix(h.Daemon, "lmtp") || strings.HasSuffix(h.Daemon, "pipe") || strings.HasSuffix(h.Daemon, "virtual") || strings.HasSuffix(h.Daemon, "local") {
		return MessageDirectionIncoming
	}

	return MessageDirectionOutbound
}

func addResultData(trackerStmts dbconn.TxPreparedStmts, time time.Time, loc postfix.RecordLocation, h parser.Header, p parser.SmtpSentStatus, resultId int64, sum postfix.Sum) error {
	if err := insertResultDataValues(trackerStmts, resultId,
		kvData{key: ResultRecipientLocalPartKey, value: p.RecipientLocalPart},
		kvData{key: ResultRecipientDomainPartKey, value: p.RecipientDomainPart},
		kvData{key: ResultOrigRecipientLocalPartKey, value: p.OrigRecipientLocalPart},
		kvData{key: ResultOrigRecipientDomainPartKey, value: p.OrigRecipientDomainPart},
		kvData{key: ResultDelayKey, value: p.Delay},
		kvData{key: ResultDelaySMTPDKey, value: p.Delays.Smtpd},
		kvData{key: ResultDelayCleanupKey, value: p.Delays.Cleanup},
		kvData{key: ResultDelayQmgrKey, value: p.Delays.Qmgr},
		kvData{key: ResultDelaySMTPKey, value: p.Delays.Smtp},
		kvData{key: ResultDSNKey, value: p.Dsn},
		kvData{key: ResultStatusKey, value: p.Status},
		kvData{key: ResultDeliveryFilenameKey, value: loc.Filename},
		kvData{key: ResultDeliveryFileLineKey, value: loc.Line},
		kvData{key: ResultDeliveryTimeKey, value: time.Unix()},
		kvData{key: ResultMessageDirectionKey, value: findMsgDirection(h)},
		kvData{key: ResultDeliveryLineChecksum, value: sum},
	); err != nil {
		return errorutil.Wrap(err)
	}

	// The relay info might be missing, and that's fine
	if p.RelayIP == nil {
		return nil
	}

	if err := insertResultDataValues(trackerStmts, resultId,
		kvData{key: ResultRelayNameKey, value: p.RelayName},
		kvData{key: ResultRelayIPKey, value: p.RelayIP},
		kvData{key: ResultRelayPortKey, value: p.RelayPort},
	); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func createResult(trackerStmts dbconn.TxPreparedStmts, r postfix.Record) (resultInfo, error) {
	//nolint:forcetypeassert
	p := r.Payload.(parser.SmtpSentStatus)

	queueId, err := findQueueIdFromQueueValue(p.Queue, trackerStmts)
	if err != nil {
		return resultInfo{}, errorutil.Wrap(err)
	}

	// Increment usage of queue, as there's one more result using it
	err = incrementQueueUsage(trackerStmts, queueId)
	if err != nil {
		return resultInfo{}, errorutil.Wrap(err)
	}

	//nolint:sqlclosecheck
	result, err := trackerStmts.Get(insertResult).Exec(queueId)
	if err != nil {
		return resultInfo{}, errorutil.Wrap(err)
	}

	resultId, err := result.LastInsertId()
	if err != nil {
		return resultInfo{}, errorutil.Wrap(err)
	}

	err = addResultData(trackerStmts, r.Time, r.Location, r.Header, p, resultId, r.Sum)
	if err != nil {
		return resultInfo{}, errorutil.Wrap(err)
	}

	return resultInfo{id: resultId, loc: r.Location}, nil
}

func mailBouncedAction(tx *sql.Tx, r postfix.Record, actionDataPair actionDataPair, trackerStmts dbconn.TxPreparedStmts) error {
	err := createMailDeliveredResult(r, trackerStmts)

	if err != nil && errors.Is(err, sql.ErrNoRows) {
		log.Warn().Msgf("Could not find queue %v for outbound e-mail, therefore ignoring it! On %v:%v",
			//nolint:forcetypeassert
			r.Payload.(parser.SmtpSentStatus).Queue, r.Location.Filename, r.Location.Line)

		return nil
	}

	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func bounceCreatedAction(tx *sql.Tx, r postfix.Record, actionDataPair actionDataPair, trackerStmts dbconn.TxPreparedStmts) error {
	//nolint:forcetypeassert
	p := r.Payload.(parser.BounceCreated)

	bounceQueueId, err := findQueueIdFromQueueValue(p.ChildQueue, trackerStmts)
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		log.Warn().Msgf("Could not find queue %v for outbound e-mail, therefore ignoring it! On %v:%v",
			p.ChildQueue, r.Location.Filename, r.Location.Line)

		return nil
	}

	if err != nil {
		return errorutil.Wrap(err)
	}

	origQueueId, err := findQueueIdFromQueueValue(p.Queue, trackerStmts)
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		log.Warn().Msgf("Could not find queue %v for outbound e-mail, therefore ignoring it! On %v:%v",
			p.Queue, r.Location.Filename, r.Location.Line)

		return nil
	}

	if err != nil {
		return errorutil.Wrap(err)
	}

	//nolint:sqlclosecheck
	_, err = trackerStmts.Get(insertQueueParenting).Exec(origQueueId, bounceQueueId, queueParentingBounceCreationType)
	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

// Mail submitted locally on the machine via sendmail is being picked up
func pickupAction(tx *sql.Tx, r postfix.Record, actionDataPair actionDataPair, trackerStmts dbconn.TxPreparedStmts) error {
	//nolint:forcetypeassert
	p := r.Payload.(parser.Pickup)

	// create a dummy connection for it, as there was no connection to it
	connectionId, err := createConnection(r, trackerStmts)
	if err != nil {
		return errorutil.Wrap(err)
	}

	// then the queue
	queueId, err := createQueue(r.Time, connectionId, p.Queue, r.Location, trackerStmts)
	if err != nil {
		return errorutil.Wrap(err)
	}

	err = insertQueueDataValues(trackerStmts, queueId,
		kvData{key: PickupUidKey, value: p.Uid},
		kvData{key: PickupSenderKey, value: p.Sender},
	)

	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

// a milter rejects a message
func milterRejectAction(tx *sql.Tx, r postfix.Record, actionDataPair actionDataPair, trackerStmts dbconn.TxPreparedStmts) error {
	// TODO: notify this rejection to someone!!!
	//nolint:forcetypeassert
	p := r.Payload.(parser.CleanupMilterReject)

	log.Warn().Msgf("Mail rejected by milter, queue: %s on %s:%v", p.Queue, r.Location.Filename, r.Location.Line)

	queueId, err := findQueueIdFromQueueValue(p.Queue, trackerStmts)

	// sometimes the milter emits the same log line more than once,
	// and in the second execution the queue is already deleted.
	// therefore the error is ignored.
	if errors.Is(err, sql.ErrNoRows) {
		return &DeletionError{Err: errorutil.Wrap(err), Loc: r.Location}
	}

	if err != nil {
		return errorutil.Wrap(err)
	}

	if _, err := tryToDeleteQueue(trackerStmts, queueId, r.Location); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func rejectAction(tx *sql.Tx, r postfix.Record, actionDataPair actionDataPair, trackerStmts dbconn.TxPreparedStmts) error {
	// TODO: Notify someone about the rejected message
	// FIXME: this is almost copy&paste from milterRejectAction!!!
	//nolint:forcetypeassert
	p := r.Payload.(parser.SmtpdReject)

	queueId, err := findQueueIdFromQueueValue(p.Queue, trackerStmts)

	if err != nil && errors.Is(err, sql.ErrNoRows) {
		log.Err(err).Msgf("Message probably already rejected with queue %s at %v", p.Queue, r.Location)
		return nil
	}

	if err != nil {
		return errorutil.Wrap(err)
	}

	if _, err := tryToDeleteQueue(trackerStmts, queueId, r.Location); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func createMessageExpiredMessage(resultId int64, loc postfix.RecordLocation, time time.Time, trackerStmts dbconn.TxPreparedStmts) error {
	if err := insertResultDataValues(trackerStmts, resultId,
		kvData{key: ResultStatusKey, value: parser.ExpiredStatus},
		kvData{key: ResultDeliveryFilenameKey, value: loc.Filename},
		kvData{key: ResultDeliveryFileLineKey, value: loc.Line},
		kvData{key: MessageExpiredTime, value: time.Unix()},
	); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func messageExpiredAction(tx *sql.Tx, r postfix.Record, actionDataPair actionDataPair, trackerStmts dbconn.TxPreparedStmts) error {
	//nolint:forcetypeassert
	p := r.Payload.(parser.QmgrMessageExpired)

	queueId, err := findQueueIdFromQueueValue(p.Queue, trackerStmts)

	if err != nil && errors.Is(err, sql.ErrNoRows) {
		log.Err(err).Msgf("Could not find queue %s at %v", p.Queue, r.Location)
		return nil
	}

	if err != nil {
		return errorutil.Wrap(err)
	}

	// Create a new result with the expired notice

	// Increment usage of queue, as there's one more result using it
	if err := incrementQueueUsage(trackerStmts, queueId); err != nil {
		return errorutil.Wrap(err)
	}

	//nolint:sqlclosecheck
	result, err := trackerStmts.Get(insertResult).Exec(queueId)
	if err != nil {
		return errorutil.Wrap(err)
	}

	resultId, err := result.LastInsertId()
	if err != nil {
		return errorutil.Wrap(err)
	}

	if err := createMessageExpiredMessage(resultId, r.Location, r.Time, trackerStmts); err != nil {
		return errorutil.Wrap(err)
	}

	if err := markResultToBeNotified(trackerStmts, resultId); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func handleInReplyToHeader(p parser.LightmeterDumpedHeader, tx *sql.Tx, r postfix.Record, trackerStmts dbconn.TxPreparedStmts) error {
	if len(p.Values) == 0 {
		return nil
	}

	queueId, err := findQueueIdFromQueueValue(r.Header, p.Queue, trackerStmts)
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		return nil
	}

	if err != nil {
		return errorutil.Wrap(err)
	}

	if err := insertQueueDataValues(trackerStmts, queueId, kvData{key: QueueInReplyToHeaderKey, value: p.Values[0]}); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func handleReferencesHeader(p parser.LightmeterDumpedHeader, tx *sql.Tx, r postfix.Record, trackerStmts dbconn.TxPreparedStmts) error {
	queueId, err := findQueueIdFromQueueValue(r.Header, p.Queue, trackerStmts)
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		return nil
	}

	if err != nil {
		return errorutil.Wrap(err)
	}

	data, err := json.Marshal(p.Values)
	if err != nil {
		return errorutil.Wrap(err)
	}

	if err := insertQueueDataValues(trackerStmts, queueId, kvData{key: QueueReferencesHeaderKey, value: data}); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func lightmeterHeaderDumpAction(tx *sql.Tx, r postfix.Record, actionDataPair actionDataPair, trackerStmts dbconn.TxPreparedStmts) error {
	//nolint:forcetypeassert
	p := r.Payload.(parser.LightmeterDumpedHeader)

	if p.Key == `In-Reply-To` {
		return handleInReplyToHeader(p, tx, r, trackerStmts)
	}

	if p.Key == `References` {
		return handleReferencesHeader(p, tx, r, trackerStmts)
	}

	return nil
}

type RelayedBounceInfos struct {
	ParserInfos parser.LightmeterRelayedBounce `json:"parser_infos"`
	RecordTime  time.Time                      `json:"record_time"`
	RecordSum   postfix.Sum                    `json:"record_sum"`
}

func lightmeterRelayedBounceAction(tx *sql.Tx, r postfix.Record, actionDataPair actionDataPair, trackerStmts dbconn.TxPreparedStmts) error {
	//nolint:forcetypeassert
	p := r.Payload.(parser.LightmeterRelayedBounce)

	queueId, err := findQueueIdFromQueueValue(p.Queue, trackerStmts)
	if err != nil {
		return errorutil.Wrap(err)
	}

	value, err := json.Marshal(RelayedBounceInfos{p, r.Time, r.Sum})
	if err != nil {
		return errorutil.Wrap(err)
	}

	if err := insertQueueDataValues(trackerStmts, queueId, kvData{key: QueueRelayedBounceJsonKey, value: value}); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}
