package tracking

import (
	"database/sql"
	"errors"
	"github.com/rs/zerolog/log"
	"gitlab.com/lightmeter/controlcenter/data"
	parser "gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
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
		case parser.DeferredStatus:
			return MailBouncedActionType, emptyActionDataPair
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
	}

	return UnsupportedActionType, emptyActionDataPair
}

func insertConnectionWithPid(tracker *Tracker, tx *sql.Tx, pidId int64) (int64, error) {
	stmt := tx.Stmt(tracker.stmts[insertConnectionOnConnection])

	defer func() {
		errorutil.MustSucceed(stmt.Close())
	}()

	result, err := stmt.Exec(pidId)
	if err != nil {
		return 0, errorutil.Wrap(err)
	}

	connectionId, err := result.LastInsertId()
	if err != nil {
		return 0, errorutil.Wrap(err)
	}

	return connectionId, nil
}

func createConnection(tracker *Tracker, tx *sql.Tx, r data.Record) (int64, error) {
	// TODO: check if there's already a connection there, as it should not be
	// in case there be, it means some message has been lost in the way
	stmt := tx.Stmt(tracker.stmts[insertPidOnConnection])

	defer func() {
		errorutil.MustSucceed(stmt.Close())
	}()

	result, err := stmt.Exec(r.Header.PID, r.Header.Host)
	if err != nil {
		return 0, errorutil.Wrap(err)
	}

	pidId, err := result.LastInsertId()
	if err != nil {
		return 0, errorutil.Wrap(err)
	}

	err = incrementPidUsage(tx, tracker.stmts, pidId)
	if err != nil {
		return 0, errorutil.Wrap(err)
	}

	connectionId, err := insertConnectionWithPid(tracker, tx, pidId)
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

	stmt := tx.Stmt(t.stmts[insertConnectionDataTwoRows])

	defer func() {
		errorutil.MustSucceed(stmt.Close())
	}()

	_, err = stmt.Exec(
		connectionId, ConnectionBeginKey, r.Time.Unix(),
		connectionId, ConnectionClientHostnameKey, payload.Host)

	if err != nil {
		return errorutil.Wrap(err)
	}

	// no IP (usually postfix sees it as "unknown"), just ignore it
	// TODO: this should be a supported use case, but I don't know what to do in this case!
	if payload.IP == nil {
		log.Warn().Msgf("Ignoring unknown IP on connection on file %v:%v", r.Location.Filename, r.Location.Line)
		return nil
	}

	stmt = tx.Stmt(t.stmts[insertConnectionData])

	defer func() {
		errorutil.MustSucceed(stmt.Close())
	}()

	_, err = stmt.Exec(connectionId, ConnectionClientIPKey, payload.IP)
	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func findConnectionIdAndUsageCounter(tx *sql.Tx, t *Tracker, h parser.Header) (int64, int, error) {
	var (
		connectionId int64
		usageCounter int
	)

	stmt := tx.Stmt(t.stmts[selectConnectionAndUsageCounterForPid])

	defer func() {
		errorutil.MustSucceed(stmt.Close())
	}()

	// find a connection entry for this
	err := stmt.QueryRow(h.Host, h.PID).Scan(&connectionId, &usageCounter)

	if err != nil {
		return 0, 0, errorutil.Wrap(err)
	}

	return connectionId, usageCounter, nil
}

func createQueue(tracker *Tracker, tx *sql.Tx, time time.Time, connectionId int64, queue string) (int64, error) {
	stmt := tx.Stmt(tracker.stmts[insertQueueForConnection])

	defer func() {
		errorutil.MustSucceed(stmt.Close())
	}()

	result, err := stmt.Exec(connectionId, queue)
	if err != nil {
		return 0, errorutil.Wrap(err)
	}

	queueId, err := result.LastInsertId()
	if err != nil {
		return 0, errorutil.Wrap(err)
	}

	stmt = tx.Stmt(tracker.stmts[insertQueueData])

	defer func() {
		errorutil.MustSucceed(stmt.Close())
	}()

	_, err = stmt.Exec(queueId, QueueBeginKey, time.Unix())

	if err != nil {
		return 0, errorutil.Wrap(err)
	}

	err = incrementConnectionUsage(tx, tracker.stmts, connectionId)
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

	connectionId, _, err := findConnectionIdAndUsageCounter(tx, tracker, r.Header)
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		log.Warn().Msgf("Connection for line %v not found", r.Location)
		return nil
	}

	if err != nil {
		return errorutil.Wrap(err)
	}

	_, err = createQueue(tracker, tx, r.Time, connectionId, p.Queue)
	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func incrementConnectionUsage(tx *sql.Tx, stmts trackerStmts, connectionId int64) error {
	stmt := tx.Stmt(stmts[incrementConnectionUsageById])

	defer func() {
		errorutil.MustSucceed(stmt.Close())
	}()

	_, err := stmt.Exec(connectionId)
	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func decrementConnectionUsage(tx *sql.Tx, stmts trackerStmts, connectionId int64) error {
	stmt := tx.Stmt(stmts[decrementConnectionUsageById])

	defer func() {
		errorutil.MustSucceed(stmt.Close())
	}()

	_, err := stmt.Exec(connectionId)
	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func getUniqueMessageIdOrCreateANewOne(tx *sql.Tx, t *Tracker, p parser.CleanupMessageAccepted) (int64, error) {
	var existingMessageId int64

	stmt := tx.Stmt(t.stmts[selectMessageIdForMessage])

	defer func() {
		errorutil.MustSucceed(stmt.Close())
	}()

	err := stmt.QueryRow(p.MessageId).Scan(&existingMessageId)
	if err == nil {
		err = incrementMessageIdUsage(tx, t.stmts, existingMessageId)

		if err != nil {
			return 0, errorutil.Wrap(err)
		}

		return existingMessageId, nil
	}

	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, errorutil.Wrap(err)
	}

	// new message-id, just insert
	stmt = tx.Stmt(t.stmts[insertMessageId])

	defer func() {
		errorutil.MustSucceed(stmt.Close())
	}()

	result, err := stmt.Exec(p.MessageId)
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

	messageidId, err := getUniqueMessageIdOrCreateANewOne(tx, tracker, p)
	if err != nil {
		return errorutil.Wrap(err)
	}

	stmt := tx.Stmt(tracker.stmts[updateQueueWithMessageId])

	defer func() {
		errorutil.MustSucceed(stmt.Close())
	}()

	_, err = stmt.Exec(messageidId, queueId)
	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func findQueueIdFromQueueValue(tx *sql.Tx, t *Tracker, h parser.Header, queue string) (int64, error) {
	var queueId int64

	stmt := tx.Stmt(t.stmts[selectQueueIdForQueue])

	defer func() {
		errorutil.MustSucceed(stmt.Close())
	}()

	err := stmt.QueryRow(
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

	stmt := tx.Stmt(tracker.stmts[insertQueueDataFourRows])

	defer func() {
		errorutil.MustSucceed(stmt.Close())
	}()

	_, err = stmt.Exec(
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
	connectionId, usageCounter, err := findConnectionIdAndUsageCounter(tx, t, r.Header)

	// it's possible for a "disconnect" not to have a "connect", if I started reading the log
	// in between the two lines. In such cases, I just ignore the line.
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		log.Warn().Msgf("Could not find a connection in log file: %v:%v", r.Location.Filename, r.Location.Line)
		return nil
	}

	if err != nil {
		return errorutil.Wrap(err)
	}

	stmt := tx.Stmt(t.stmts[insertConnectionData])

	defer func() {
		errorutil.MustSucceed(stmt.Close())
	}()

	_, err = stmt.Exec(
		connectionId, ConnectionEndKey, r.Time.Unix())

	if err != nil {
		return errorutil.Wrap(err)
	}

	if usageCounter > 0 {
		// Cannot delete the connection yet as there are queues using it!
		return nil
	}

	err = deleteConnection(tx, t.stmts, connectionId)
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
	resultInfo, err := createResult(t, tx, r)
	if err != nil {
		return errorutil.Wrap(err)
	}

	err = markResultToBeNotified(t, tx, resultInfo)
	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func mailSentAction(t *Tracker, tx *sql.Tx, r data.Record, actionDataPair actionDataPair) error {
	// Check if message has been forwarded to the an internal relay
	p := r.Payload.(parser.SmtpSentStatus)

	e, messageQueuedInternally := p.ExtraMessagePayload.(parser.SmtpStatusExtraMessageSentQueued)

	// delivery to the next relay outside of the system
	if !messageQueuedInternally {
		// not internally queued
		err := createMailDeliveredResult(t, tx, r)

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

	newQueueId, err := findQueueIdFromQueueValue(tx, t, r.Header, e.Queue)
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		log.Warn().Msgf("Queue has been lost forever and will be ignored: %v, on %v:%v", e.Queue, r.Location.Filename, r.Location.Line)
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
		return nil
	}

	if err != nil {
		return errorutil.Wrap(err)
	}

	// this is an e-mail that postfix sends to itself before trying to deliver.
	// As it's moved to another queue to be delivered, we queue the original and
	// the newly created queue
	stmt := tx.Stmt(t.stmts[insertQueueParenting])

	defer func() {
		errorutil.MustSucceed(stmt.Close())
	}()

	_, err = stmt.Exec(
		origQueueId, newQueueId, queueParentingRelayType)
	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func markResultToBeNotified(tracker *Tracker, tx *sql.Tx, resultInfo resultInfo) error {
	stmt := tx.Stmt(tracker.stmts[insertNotificationQueue])

	defer func() {
		errorutil.MustSucceed(stmt.Close())
	}()

	_, err := stmt.Exec(
		resultInfo.id, resultInfo.loc.Filename, resultInfo.loc.Line)

	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func commitAction(tracker *Tracker, tx *sql.Tx, r data.Record, actionDataPair actionDataPair) error {
	p := r.Payload.(parser.QmgrRemoved)

	queueId, err := findQueueIdFromQueueValue(tx, tracker, r.Header, p.Queue)

	if err != nil && errors.Is(err, sql.ErrNoRows) {
		log.Warn().Msgf("Could not find queue %v for outbound e-mail, therefore ignoring it! On %v:%v", p.Queue, r.Location.Filename, r.Location.Line)
		return nil
	}

	if err != nil {
		return errorutil.Wrap(err)
	}

	stmt := tx.Stmt(tracker.stmts[insertQueueData])

	defer func() {
		errorutil.MustSucceed(stmt.Close())
	}()

	_, err = stmt.Exec(queueId, QueueEndKey, r.Time.Unix())
	if err != nil {
		return errorutil.Wrap(err)
	}

	_, err = tryToDeleteQueue(tx, tracker.stmts, queueId, r.Location)
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

	stmt := tx.Stmt(tracker.stmts[insertResultData15Rows])

	defer func() {
		errorutil.MustSucceed(stmt.Close())
	}()

	_, err := stmt.Exec(
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

	stmt = tx.Stmt(tracker.stmts[insertResultData3Rows])

	defer func() {
		errorutil.MustSucceed(stmt.Close())
	}()

	_, err = stmt.Exec(
		resultId, ResultRelayNameKey, p.RelayName,
		resultId, ResultRelayIPKey, p.RelayIP,
		resultId, ResultRelayPortKey, p.RelayPort,
	)

	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func createResult(tracker *Tracker, tx *sql.Tx, r data.Record) (resultInfo, error) {
	p := r.Payload.(parser.SmtpSentStatus)

	queueId, err := findQueueIdFromQueueValue(tx, tracker, r.Header, p.Queue)
	if err != nil {
		return resultInfo{}, errorutil.Wrap(err)
	}

	// Increment usage of queue, as there's one more result using it
	err = incrementQueueUsage(tx, tracker.stmts, queueId)
	if err != nil {
		return resultInfo{}, errorutil.Wrap(err)
	}

	stmt := tx.Stmt(tracker.stmts[insertResult])

	defer func() {
		errorutil.MustSucceed(stmt.Close())
	}()

	result, err := stmt.Exec(queueId)
	if err != nil {
		return resultInfo{}, errorutil.Wrap(err)
	}

	resultId, err := result.LastInsertId()
	if err != nil {
		return resultInfo{}, errorutil.Wrap(err)
	}

	err = addResultData(tracker, tx, r.Time, r.Location, r.Header, p, resultId)
	if err != nil {
		return resultInfo{}, errorutil.Wrap(err)
	}

	return resultInfo{id: resultId, loc: r.Location}, nil
}

func mailBouncedAction(tracker *Tracker, tx *sql.Tx, r data.Record, actionDataPair actionDataPair) error {
	err := createMailDeliveredResult(tracker, tx, r)

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

func bounceCreatedAction(tracker *Tracker, tx *sql.Tx, r data.Record, actionDataPair actionDataPair) error {
	p := r.Payload.(parser.BounceCreated)

	bounceQueueId, err := findQueueIdFromQueueValue(tx, tracker, r.Header, p.ChildQueue)
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		log.Warn().Msgf("Could not find queue %v for outbound e-mail, therefore ignoring it! On %v:%v",
			p.ChildQueue, r.Location.Filename, r.Location.Line)

		return nil
	}

	if err != nil {
		return errorutil.Wrap(err)
	}

	origQueueId, err := findQueueIdFromQueueValue(tx, tracker, r.Header, p.Queue)
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		log.Warn().Msgf("Could not find queue %v for outbound e-mail, therefore ignoring it! On %v:%v",
			p.Queue, r.Location.Filename, r.Location.Line)

		return nil
	}

	if err != nil {
		return errorutil.Wrap(err)
	}

	stmt := tx.Stmt(tracker.stmts[insertQueueParenting])

	defer func() {
		errorutil.MustSucceed(stmt.Close())
	}()

	_, err = stmt.Exec(origQueueId, bounceQueueId, queueParentingBounceCreationType)
	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

// Mail submitted locally on the machine via sendmail is being picked up
func pickupAction(t *Tracker, tx *sql.Tx, r data.Record, actionDataPair actionDataPair) error {
	p := r.Payload.(parser.Pickup)

	// create a dummy connection for it, as there was no connection to it
	connectionId, err := createConnection(t, tx, r)
	if err != nil {
		return errorutil.Wrap(err)
	}

	// then the queue
	queueId, err := createQueue(t, tx, r.Time, connectionId, p.Queue)
	if err != nil {
		return errorutil.Wrap(err)
	}

	stmt := tx.Stmt(t.stmts[insertQueueDataTwoRows])

	defer func() {
		errorutil.MustSucceed(stmt.Close())
	}()

	_, err = stmt.Exec(queueId, PickupUidKey, p.Uid, queueId, PickupSenderKey, p.Sender)
	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}
