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
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"strings"
	"time"
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
)

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

var emptyActionDataPair = actionDataPair{connectionActionData: nil, resultActionData: nil}

func actionTypeForRecord(r postfix.Record) (ActionType, actionDataPair) {
	// FIXME: this function will get uglier over time :-(
	switch p := r.Payload.(type) {
	case parser.SmtpSentStatus:
		switch p.Status {
		case parser.BouncedStatus:
			return MailBouncedActionType, emptyActionDataPair
		case parser.SentStatus:
			return MailSentActionType, emptyActionDataPair
		case parser.DeferredStatus:
			return MailBouncedActionType, emptyActionDataPair
		case parser.ExpiredStatus:
			fallthrough
		case parser.ReturnedStatus:
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
	}

	return UnsupportedActionType, emptyActionDataPair
}

func insertConnectionWithPid(trackerStmts dbconn.TxPreparedStmts, pidId int64) (int64, error) {
	result, err := trackerStmts.S[insertConnectionOnConnection].Exec(pidId)
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
	result, err := trackerStmts.S[insertPidOnConnection].Exec(pid, host)
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

	err := trackerStmts.S[selectPidForPidAndHost].QueryRow(pid, host).Scan(&pidId)
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

	_, err = trackerStmts.S[insertConnectionDataFourRows].Exec(
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

	_, err = trackerStmts.S[insertConnectionData].Exec(connectionId, ConnectionClientIPKey, payload.IP)
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
	err := trackerStmts.S[selectConnectionAndUsageCounterForPid].QueryRow(h.Host, h.PID).Scan(&connectionId, &usageCounter)

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
		if _, err := stmts.S[insertQueueData].Exec(queueId, v.key, v.value); err != nil {
			return errorutil.Wrap(err)
		}
	}

	return nil
}

func createQueue(time time.Time, connectionId int64, queue string, loc postfix.RecordLocation, trackerStmts dbconn.TxPreparedStmts) (int64, error) {
	result, err := trackerStmts.S[insertQueueForConnection].Exec(connectionId, queue)
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

	_, err = createQueue(r.Time, connectionId, p.Queue, r.Location, trackerStmts)
	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func incrementConnectionUsage(stmts dbconn.TxPreparedStmts, connectionId int64) error {
	_, err := stmts.S[incrementConnectionUsageById].Exec(connectionId)
	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func decrementConnectionUsage(stmts dbconn.TxPreparedStmts, connectionId int64) error {
	_, err := stmts.S[decrementConnectionUsageById].Exec(connectionId)
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
		queueId, err := findQueueIdFromQueueValue(r.Header, p.Queue, trackerStmts)
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

func findQueueIdFromQueueValue(h parser.Header, queue string, trackerStmts dbconn.TxPreparedStmts) (int64, error) {
	var queueId int64

	err := trackerStmts.S[selectQueueIdForQueue].QueryRow(
		h.Host, queue).Scan(&queueId)

	if err != nil {
		return 0, errorutil.Wrap(err, "No queue id for queue: ", queue)
	}

	return queueId, nil
}

func mailQueuedAction(tx *sql.Tx, r postfix.Record, actionDataPair actionDataPair, trackerStmts dbconn.TxPreparedStmts) error {
	// I have the queue id and need to set the e-mail sender, size and nrcpt
	//nolint:forcetypeassert
	p := r.Payload.(parser.QmgrMailQueued)

	queueId, err := findQueueIdFromQueueValue(r.Header, p.Queue, trackerStmts)
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

	_, err = trackerStmts.S[insertConnectionData].Exec(connectionId, ConnectionEndKey, r.Time.Unix())
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

func mailSentAction(tx *sql.Tx, r postfix.Record, actionDataPair actionDataPair, trackerStmts dbconn.TxPreparedStmts) error {
	// Check if message has been forwarded to the an internal relay
	//nolint:forcetypeassert
	p := r.Payload.(parser.SmtpSentStatus)

	e, messageQueuedInternally := p.ExtraMessagePayload.(parser.SmtpSentStatusExtraMessageSentQueued)

	// delivery to the next relay outside of the system
	if !messageQueuedInternally {
		// not internally queued
		err := createMailDeliveredResult(r, trackerStmts)

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

		return nil
	}

	newQueueId, err := findQueueIdFromQueueValue(r.Header, e.Queue, trackerStmts)
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		log.Warn().Msgf("Queue has been lost forever and will be ignored: %v, on %v:%v at %v", e.Queue, r.Location.Filename, r.Location.Line, r.Time)
		return nil
	}

	if err != nil {
		return errorutil.Wrap(err)
	}

	origQueueId, err := findQueueIdFromQueueValue(r.Header, p.Queue, trackerStmts)

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

	// this is an e-mail that postfix sends to itself before trying to deliver.
	// As it's moved to another queue to be delivered, we queue the original and
	// the newly created queue
	_, err = trackerStmts.S[insertQueueParenting].Exec(origQueueId, newQueueId, queueParentingRelayType)
	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func markResultToBeNotified(trackerStmts dbconn.TxPreparedStmts, resultId int64) error {
	_, err := trackerStmts.S[insertNotificationQueue].Exec(resultId)
	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func commitAction(tx *sql.Tx, r postfix.Record, actionDataPair actionDataPair, trackerStmts dbconn.TxPreparedStmts) error {
	//nolint:forcetypeassert
	p := r.Payload.(parser.QmgrRemoved)

	queueId, err := findQueueIdFromQueueValue(r.Header, p.Queue, trackerStmts)

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

func addResultData(trackerStmts dbconn.TxPreparedStmts, time time.Time, loc postfix.RecordLocation, h parser.Header, p parser.SmtpSentStatus, resultId int64) error {
	direction := func() MessageDirection {
		if strings.HasSuffix(h.Daemon, "lmtp") || strings.HasSuffix(h.Daemon, "pipe") || strings.HasSuffix(h.Daemon, "virtual") {
			return MessageDirectionIncoming
		}

		return MessageDirectionOutbound
	}()

	_, err := trackerStmts.S[insertResultData15Rows].Exec(
		resultId, ResultRecipientLocalPartKey, p.RecipientLocalPart,
		resultId, ResultRecipientDomainPartKey, p.RecipientDomainPart,
		resultId, ResultOrigRecipientLocalPartKey, p.OrigRecipientLocalPart,
		resultId, ResultOrigRecipientDomainPartKey, p.OrigRecipientDomainPart,
		resultId, ResultDelayKey, p.Delay,
		resultId, ResultDelaySMTPDKey, p.Delays.Smtpd,
		resultId, ResultDelayCleanupKey, p.Delays.Cleanup,
		resultId, ResultDelayQmgrKey, p.Delays.Qmgr,
		resultId, ResultDelaySMTPKey, p.Delays.Smtp,
		resultId, ResultDSNKey, p.Dsn,
		resultId, ResultStatusKey, p.Status,
		resultId, ResultDeliveryFilenameKey, loc.Filename,
		resultId, ResultDeliveryFileLineKey, loc.Line,
		resultId, ResultDeliveryTimeKey, time.Unix(),
		resultId, ResultMessageDirectionKey, direction,
	)

	if err != nil {
		return errorutil.Wrap(err)
	}

	// The relay info might be missing, and that's fine
	if p.RelayIP == nil {
		return nil
	}

	_, err = trackerStmts.S[insertResultData3Rows].Exec(
		resultId, ResultRelayNameKey, p.RelayName,
		resultId, ResultRelayIPKey, p.RelayIP,
		resultId, ResultRelayPortKey, p.RelayPort,
	)

	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func createResult(trackerStmts dbconn.TxPreparedStmts, r postfix.Record) (resultInfo, error) {
	//nolint:forcetypeassert
	p := r.Payload.(parser.SmtpSentStatus)

	queueId, err := findQueueIdFromQueueValue(r.Header, p.Queue, trackerStmts)
	if err != nil {
		return resultInfo{}, errorutil.Wrap(err)
	}

	// Increment usage of queue, as there's one more result using it
	err = incrementQueueUsage(trackerStmts, queueId)
	if err != nil {
		return resultInfo{}, errorutil.Wrap(err)
	}

	result, err := trackerStmts.S[insertResult].Exec(queueId)
	if err != nil {
		return resultInfo{}, errorutil.Wrap(err)
	}

	resultId, err := result.LastInsertId()
	if err != nil {
		return resultInfo{}, errorutil.Wrap(err)
	}

	err = addResultData(trackerStmts, r.Time, r.Location, r.Header, p, resultId)
	if err != nil {
		return resultInfo{}, errorutil.Wrap(err)
	}

	return resultInfo{id: resultId, loc: r.Location}, nil
}

func mailBouncedAction(tx *sql.Tx, r postfix.Record, actionDataPair actionDataPair, trackerStmts dbconn.TxPreparedStmts) error {
	err := createMailDeliveredResult(r, trackerStmts)

	if err != nil && errors.Is(err, sql.ErrNoRows) {
		log.Warn().Msgf("Could not find queue %v for outbound e-mail, therefore ignoring it! On %v:%v",
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

	bounceQueueId, err := findQueueIdFromQueueValue(r.Header, p.ChildQueue, trackerStmts)
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		log.Warn().Msgf("Could not find queue %v for outbound e-mail, therefore ignoring it! On %v:%v",
			p.ChildQueue, r.Location.Filename, r.Location.Line)

		return nil
	}

	if err != nil {
		return errorutil.Wrap(err)
	}

	origQueueId, err := findQueueIdFromQueueValue(r.Header, p.Queue, trackerStmts)
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		log.Warn().Msgf("Could not find queue %v for outbound e-mail, therefore ignoring it! On %v:%v",
			p.Queue, r.Location.Filename, r.Location.Line)

		return nil
	}

	if err != nil {
		return errorutil.Wrap(err)
	}

	_, err = trackerStmts.S[insertQueueParenting].Exec(origQueueId, bounceQueueId, queueParentingBounceCreationType)
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

	queueId, err := findQueueIdFromQueueValue(r.Header, p.Queue, trackerStmts)

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

	queueId, err := findQueueIdFromQueueValue(r.Header, p.Queue, trackerStmts)

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
	if _, err := trackerStmts.S[insertResultData4Rows].Exec(
		resultId, ResultStatusKey, parser.ExpiredStatus,
		resultId, ResultDeliveryFilenameKey, loc.Filename,
		resultId, ResultDeliveryFileLineKey, loc.Line,
		resultId, MessageExpiredTime, time.Unix(),
	); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func messageExpiredAction(tx *sql.Tx, r postfix.Record, actionDataPair actionDataPair, trackerStmts dbconn.TxPreparedStmts) error {
	//nolint:forcetypeassert
	p := r.Payload.(parser.QmgrMessageExpired)

	queueId, err := findQueueIdFromQueueValue(r.Header, p.Queue, trackerStmts)

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

	result, err := trackerStmts.S[insertResult].Exec(queueId)
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
