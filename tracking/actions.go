package tracking

import (
	"database/sql"
	"errors"
	"gitlab.com/lightmeter/controlcenter/data"
	parser "gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"log"
	"strings"
	"time"
)

var emptyActionDataPair = actionDataPair{connectionActionData: nil, resultActionData: nil}

func actionTypeForRecord(r data.Record) (ActionType, actionDataPair) {
	// FIXME: this function will get uglier over time :-(
	switch p := r.Payload.(type) {
	case parser.SmtpSentStatus:
		switch p.Status {
		case parser.BouncedStatus:
			return MailBouncedActionType, emptyActionDataPair
		case parser.SentStatus:
			return MailSentActionType, emptyActionDataPair
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
	}

	return UnsupportedActionType, emptyActionDataPair
}

// TODO: prepare all queries during application startup

func createConnection(tracker *Tracker, tx *sql.Tx, r data.Record) (int64, error) {
	// TODO: check if there's already a connection there, as it should not be
	// in case there be, it means some message has been lost in the way
	result, err := tx.Stmt(tracker.stmts[insertPidOnConnection]).Exec(r.Header.PID, r.Header.Host)
	if err != nil {
		return 0, errorutil.Wrap(err)
	}

	pidId, err := result.LastInsertId()
	if err != nil {
		return 0, errorutil.Wrap(err)
	}

	result, err = tx.Stmt(tracker.stmts[insertConnectionOnConnection]).Exec(pidId)
	if err != nil {
		return 0, errorutil.Wrap(err)
	}

	connectionId, err := result.LastInsertId()
	if err != nil {
		return 0, errorutil.Wrap(err)
	}

	return connectionId, nil
}

func connectAction(t *Tracker, tx *sql.Tx, r data.Record, actionDataPair actionDataPair) error {
	// TODO: check if there's already a connection there, as it should not be
	// in case there be, it means some message has been lost in the way
	connectionId, err := createConnection(t, tx, r)
	if err != nil {
		return errorutil.Wrap(err)
	}

	// TODO: there might be other payloads about connection, so this cast is not always safe
	payload := r.Payload.(parser.SmtpdConnect)

	_, err = tx.Stmt(t.stmts[insertConnectionDataTwoRows]).Exec(
		connectionId, ConnectionBeginKey, r.Time.Unix(),
		connectionId, ConnectionClientHostnameKey, payload.Host)

	if err != nil {
		return errorutil.Wrap(err)
	}

	// no IP (usually postfix sees it as "unknown"), just ignore it
	if payload.IP == nil {
		return nil
	}

	_, err = tx.Stmt(t.stmts[insertConnectionData]).Exec(connectionId, ConnectionClientIPKey, payload.IP)
	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func findConnectionId(tx *sql.Tx, t *Tracker, h parser.Header) (int64, error) {
	var connectionId int64

	// find a connection entry for this
	err := tx.Stmt(t.stmts[selectConnectionForPid]).QueryRow(h.Host, h.PID).Scan(&connectionId)

	if err != nil {
		return 0, errorutil.Wrap(err)
	}

	return connectionId, nil
}

func createQueue(tracker *Tracker, tx *sql.Tx, time time.Time, connectionId int64, queue string) (int64, error) {
	result, err := tx.Stmt(tracker.stmts[insertQueueForConnection]).Exec(connectionId, queue)
	if err != nil {
		return 0, errorutil.Wrap(err)
	}

	queueId, err := result.LastInsertId()
	if err != nil {
		return 0, errorutil.Wrap(err)
	}

	_, err = tx.Stmt(tracker.stmts[insertQueueData]).Exec(queueId, QueueBeginKey, time.Unix())

	if err != nil {
		return 0, errorutil.Wrap(err)
	}

	return queueId, nil
}

// assign a queue, just created.
// find the connection with a given pid, and append the queue to the connection
// TODO: handle cases where there are more than one queue for a given pid.
// (but first, confirm whether that's possible)
// In such cases, QueryRow() below will return multiple rows

func cloneAction(tracker *Tracker, tx *sql.Tx, r data.Record, actionDataPair actionDataPair) error {
	p := r.Payload.(parser.SmtpdMailAccepted)

	connectionId, err := findConnectionId(tx, tracker, r.Header)
	if err != nil {
		return errorutil.Wrap(err)
	}

	_, err = createQueue(tracker, tx, r.Time, connectionId, p.Queue)
	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func getUniqueMessageId(tx *sql.Tx, t *Tracker, p parser.CleanupMessageAccepted) (int64, error) {
	var existingMessageId int64

	err := tx.Stmt(t.stmts[selectMessageIdForMessage]).QueryRow(p.MessageId).Scan(&existingMessageId)
	if err == nil {
		return existingMessageId, nil
	}

	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, errorutil.Wrap(err)
	}

	// new message-id, just insert
	result, err := tx.Stmt(t.stmts[insertMessageId]).Exec(p.MessageId)
	if err != nil {
		return 0, errorutil.Wrap(err)
	}

	messageidId, err := result.LastInsertId()
	if err != nil {
		return 0, errorutil.Wrap(err)
	}

	return messageidId, nil
}

// associate a queue to a message-id
func cleanupProcessingAction(tracker *Tracker, tx *sql.Tx, r data.Record, actionDataPair actionDataPair) error {
	p := r.Payload.(parser.CleanupMessageAccepted)

	queueId, err := func() (int64, error) {
		queueId, err := findQueueIdFromQueueValue(tx, tracker, r.Header, p.Queue)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return 0, errorutil.Wrap(err)
		}

		if err == nil {
			return queueId, nil
		}

		// Create a dummy connection with no data, meaning it's been generated by the server itself, not via SMTP
		connectionId, err := createConnection(tracker, tx, r)
		if err != nil {
			return 0, errorutil.Wrap(err)
		}

		// Then a queue for it

		queueId, err = createQueue(tracker, tx, r.Time, connectionId, p.Queue)
		if err != nil {
			return 0, errorutil.Wrap(err)
		}

		return queueId, nil
	}()

	if err != nil {
		return errorutil.Wrap(err)
	}

	messageidId, err := getUniqueMessageId(tx, tracker, p)
	if err != nil {
		return errorutil.Wrap(err)
	}

	_, err = tx.Stmt(tracker.stmts[updateQueueWithMessageId]).Exec(messageidId, queueId)
	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func findQueueIdFromQueueValue(tx *sql.Tx, t *Tracker, h parser.Header, queue string) (int64, error) {
	var queueId int64

	err := tx.Stmt(t.stmts[selectQueueIdForQueue]).QueryRow(
		h.Host, queue).Scan(&queueId)

	if err != nil {
		return 0, errorutil.Wrap(err, "No queue id for queue: ", queue)
	}

	return queueId, nil
}

func mailQueuedAction(tracker *Tracker, tx *sql.Tx, r data.Record, actionDataPair actionDataPair) error {
	// I have the queue id and need to set the e-mail sender, size and nrcpt
	p := r.Payload.(parser.QmgrMailQueued)

	queueId, err := findQueueIdFromQueueValue(tx, tracker, r.Header, p.Queue)
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		// started reading the logs when a queue is referenced, but not known (it was on a previous and unknown log)
		// just ignore it.
		return nil
	}

	if err != nil {
		return errorutil.Wrap(err)
	}

	_, err = tx.Stmt(tracker.stmts[insertQueueDataFourRows]).Exec(
		queueId, QueueSenderLocalPartKey, p.SenderLocalPart,
		queueId, QueueSenderDomainPartKey, p.SenderDomainPart,
		queueId, QueueOriginalMessageSizeKey, p.Size,
		queueId, QueueNRCPTKey, p.Nrcpt,
	)

	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func disconnectAction(t *Tracker, tx *sql.Tx, r data.Record, actionDataPair actionDataPair) error {
	connectionId, err := findConnectionId(tx, t, r.Header)

	// it's possible for a "disconnect" not to have a "connect", if I started reading the log
	// in between the two lines. In such cases, I just ignore the line.
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		log.Println("Could not find a connection in log file:", r.Location.Filename, ":", r.Location.Line)
		return nil
	}

	if err != nil {
		return errorutil.Wrap(err)
	}

	_, err = tx.Stmt(t.stmts[insertConnectionData]).Exec(
		connectionId, ConnectionEndKey, r.Time.Unix())

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

func createMailDeliveredResult(t *Tracker, tx *sql.Tx, r data.Record) error {
	err := createResult(t, tx, r)
	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func mailSentAction(t *Tracker, tx *sql.Tx, r data.Record, actionDataPair actionDataPair) error {
	// Check if message has been forwarded to the an internal relay
	p := r.Payload.(parser.SmtpSentStatus)

	e, messageQueuedInternally := p.ExtraMessagePayload.(parser.SmtpStatusExtraMessageSentQueued)

	if !messageQueuedInternally {
		// not internally queued
		err := createMailDeliveredResult(t, tx, r)

		if err != nil && errors.Is(err, sql.ErrNoRows) {
			// sometimes a queue has been created in a point earlier than what's known by us. Just ignore it then
			// TODO: postfix can have very long living queues (that are active for many days)
			// and can use such queue for delivering many e-mails.
			// Right now we are not notifying any of those intermediate e-mails, which might not be desisable
			// More investigation is needed
			log.Println("Could not find queue", p.Queue, "for outbound sent e-mail, therefore ignoring it")
			return nil
		}

		if err != nil {
			return errorutil.Wrap(err)
		}

		return nil
	}

	newQueueId, err := findQueueIdFromQueueValue(tx, t, r.Header, e.Queue)
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		log.Println("Queue has been lost forever and will be ignored:", e.Queue)
		return nil
	}

	if err != nil {
		return errorutil.Wrap(err)
	}

	origQueueId, err := findQueueIdFromQueueValue(tx, t, r.Header, p.Queue)

	// TODO: this block is copy&pasted many times! It should be refactored!
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		// sometimes a queue has been created in a point earlier than what's known by us. Just ignore it then
		// TODO: postfix can have very long living queues (that are active for many days)
		// and can use such queue for delivering many e-mails.
		// Right now we are not notifying any of those intermediate e-mails, which might not be desisable
		// More investigation is needed
		log.Println("Could not find queue", p.Queue, "for outbound sent e-mail, therefore ignoring it")
		return nil
	}

	if err != nil {
		return errorutil.Wrap(err)
	}

	// this is an e-mail that postfix sends to itself before trying to deliver.
	// As it's moved to another queue to be delivered, we queue the original and
	// the newly created queue
	_, err = tx.Stmt(t.stmts[insertQueueParenting]).Exec(
		origQueueId, newQueueId, queueParentingRelayType)
	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func markQueueToBeNotified(tracker *Tracker, tx *sql.Tx, queueInfo queueInfo) error {
	_, err := tx.Stmt(tracker.stmts[insertNotificationQueue]).Exec(
		queueInfo.queueId, queueInfo.loc.Filename, queueInfo.loc.Line)

	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func commitAction(tracker *Tracker, tx *sql.Tx, r data.Record, actionDataPair actionDataPair) error {
	p := r.Payload.(parser.QmgrRemoved)

	queueId, err := findQueueIdFromQueueValue(tx, tracker, r.Header, p.Queue)

	// TODO: this block is copy&pasted many times! It should be refactored!
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		// sometimes a queue has been created in a point earlier than what's known by us. Just ignore it then
		// TODO: postfix can have very long living queues (that are active for many days)
		// and can use such queue for delivering many e-mails.
		// Right now we are not notifying any of those intermediate e-mails, which might not be desisable
		// More investigation is needed
		log.Println("Could not find queue", p.Queue, "for outbound sent e-mail, therefore ignoring it")
		return nil
	}

	if err != nil {
		return errorutil.Wrap(err)
	}

	_, err = tx.Stmt(tracker.stmts[insertQueueData]).Exec(queueId, QueueEndKey, r.Time.Unix())

	if err != nil {
		return errorutil.Wrap(err)
	}

	var newQueueId int64

	err = tx.Stmt(tracker.stmts[selectNewQueueFromParenting]).QueryRow(queueId).Scan(&newQueueId)

	if err != nil && errors.Is(err, sql.ErrNoRows) {
		err = markQueueToBeNotified(tracker, tx, queueInfo{
			queueId: queueId,
			loc:     data.RecordLocation{Line: r.Location.Line, Filename: r.Location.Filename},
		})

		if err != nil {
			return errorutil.Wrap(err)
		}

		return nil
	}

	if err != nil {
		return errorutil.Wrap(err)
	}

	var newQueue string

	err = tx.Stmt(tracker.stmts[selectQueueById]).QueryRow(newQueueId).Scan(&newQueue)
	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func addResultData(tracker *Tracker, tx *sql.Tx, time time.Time, loc data.RecordLocation, h parser.Header, p parser.SmtpSentStatus, resultId int64) error {
	direction := func() MessageDirection {
		if strings.HasSuffix(h.Daemon, "lmtp") {
			return MessageDirectionIncoming
		}

		return MessageDirectionOutbound
	}()

	_, err := tx.Stmt(tracker.stmts[insertResultData15Rows]).Exec(
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

	_, err = tx.Stmt(tracker.stmts[insertResultData3Rows]).Exec(
		resultId, ResultRelayNameKey, p.RelayName,
		resultId, ResultRelayIPKey, p.RelayIP,
		resultId, ResultRelayPortKey, p.RelayPort,
	)

	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func createResult(tracker *Tracker, tx *sql.Tx, r data.Record) error {
	p := r.Payload.(parser.SmtpSentStatus)

	queueId, err := findQueueIdFromQueueValue(tx, tracker, r.Header, p.Queue)
	if err != nil {
		return errorutil.Wrap(err)
	}

	result, err := tx.Stmt(tracker.stmts[insertResult]).Exec(queueId)
	if err != nil {
		return errorutil.Wrap(err)
	}

	resultId, err := result.LastInsertId()
	if err != nil {
		return errorutil.Wrap(err)
	}

	err = addResultData(tracker, tx, r.Time, r.Location, r.Header, p, resultId)
	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func mailBouncedAction(tracker *Tracker, tx *sql.Tx, r data.Record, actionDataPair actionDataPair) error {
	err := createResult(tracker, tx, r)

	// TODO: this block is copy&pasted many times! It should be refactored!
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		// sometimes a queue has been created in a point earlier than what's known by us. Just ignore it then
		// TODO: postfix can have very long living queues (that are active for many days)
		// and can use such queue for delivering many e-mails.
		// Right now we are not notifying any of those intermediate e-mails, which might not be desisable
		// More investigation is needed
		log.Println("Could not find queue", r.Payload.(parser.SmtpSentStatus).Queue, "for outbound sent e-mail, therefore ignoring it")
		return nil
	}

	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func bounceCreatedAction(tracker *Tracker, tx *sql.Tx, r data.Record, actionDataPair actionDataPair) error {
	p := r.Payload.(parser.BounceCreated)

	bounceQueueId, err := findQueueIdFromQueueValue(tx, tracker, r.Header, p.ChildQueue)
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		// sometimes a queue has been created in a point earlier than what's known by us. Just ignore it then
		// TODO: postfix can have very long living queues (that are active for many days)
		// and can use such queue for delivering many e-mails.
		// Right now we are not notifying any of those intermediate e-mails, which might not be desisable
		// More investigation is needed
		log.Println("Could not find queue", p.ChildQueue, "for outbound sent e-mail, therefore ignoring it")
		return nil
	}

	if err != nil {
		return errorutil.Wrap(err)
	}

	origQueueId, err := findQueueIdFromQueueValue(tx, tracker, r.Header, p.Queue)
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		// sometimes a queue has been created in a point earlier than what's known by us. Just ignore it then
		// TODO: postfix can have very long living queues (that are active for many days)
		// and can use such queue for delivering many e-mails.
		// Right now we are not notifying any of those intermediate e-mails, which might not be desisable
		// More investigation is needed
		log.Println("Could not find queue", p.Queue, "for outbound sent e-mail, therefore ignoring it")
		return nil
	}

	if err != nil {
		return errorutil.Wrap(err)
	}

	_, err = tx.Stmt(tracker.stmts[insertQueueParenting]).Exec(origQueueId, bounceQueueId, queueParentingBounceCreationType)
	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}
