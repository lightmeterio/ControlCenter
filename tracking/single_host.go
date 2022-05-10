// SPDX-FileCopyrightText: 2022 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package tracking

import (
	"database/sql"
	"errors"
	"time"

	"github.com/rs/zerolog/log"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/pkg/postfix"
	parser "gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
)

type SingleNodeTypeHandler struct {
}

func (h SingleNodeTypeHandler) CreateQueue(t time.Time, connectionId int64, queue string, location postfix.RecordLocation, trackerStmts dbconn.TxPreparedStmts) (int64, error) {
	return createQueue(t, connectionId, queue, location, trackerStmts)
}

func (h *SingleNodeTypeHandler) FindQueue(queue string, r postfix.Record, stmts dbconn.TxPreparedStmts) (int64, error) {
	return findQueueIdFromQueueValue(queue, stmts)
}

func (h *SingleNodeTypeHandler) HandleMailSentAction(tx *sql.Tx, r postfix.Record, p parser.SmtpSentStatus, trackerStmts dbconn.TxPreparedStmts) error {
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

	newQueueId, err := findQueueIdFromQueueValue(e.Queue, trackerStmts)
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		log.Warn().Msgf("Queue has been lost forever and will be ignored: %v, on %v:%v at %v", e.Queue, r.Location.Filename, r.Location.Line, r.Time)
		return nil
	}

	if err != nil {
		return errorutil.Wrap(err)
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

	// this is an e-mail that postfix sends to itself before trying to deliver.
	// As it's moved to another queue to be delivered, we queue the original and
	// the newly created queue
	//nolint:sqlclosecheck
	_, err = trackerStmts.Get(insertQueueParenting).Exec(origQueueId, newQueueId, queueParentingRelayType)
	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}
