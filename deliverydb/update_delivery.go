// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package deliverydb

import (
	"database/sql"
	"github.com/rs/zerolog/log"
	parser "gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser"
	"gitlab.com/lightmeter/controlcenter/tracking"
	"gitlab.com/lightmeter/controlcenter/util/emailutil"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"time"
)

type delivery struct {
	id, ts int64
}

func updateDeliveryWithBounceInfo(tx *sql.Tx, rb tracking.RelayedBounceInfos) error {
	senderU, senderD, err := emailutil.Split(rb.ParserInfos.Sender)
	if err != nil {
		return errorutil.Wrap(err)
	}

	recipientU, recipientD, err := emailutil.Split(rb.ParserInfos.Recipient)
	if err != nil {
		return errorutil.Wrap(err)
	}

	var deliveries []delivery

	rows, err := tx.Query(stmtsText[selectDeliveries],
		sql.Named("sender_user", senderU),
		sql.Named("sender_domain", senderD),
		sql.Named("recipient_user", recipientU),
		sql.Named("recipient_domain", recipientD),
		sql.Named("an_hour_ago", rb.RecordTime.Add(-1*time.Hour).Unix()),
	)

	if err != nil {
		return errorutil.Wrap(err)
	}

	defer errorutil.UpdateErrorFromCloser(rows, &err)

	deliveries = []delivery{}

	for rows.Next() {
		var d delivery

		if err := rows.Scan(&d.id, &d.ts); err != nil {
			return errorutil.Wrap(err)
		}

		deliveries = append(deliveries, d)
	}

	if err := rows.Err(); err != nil {
		return errorutil.Wrap(err)
	}

	if len(deliveries) != 1 {
		log.Warn().Msgf("Relayed bounce updates %d lines (%s => %s)", len(deliveries), rb.ParserInfos.Sender, rb.ParserInfos.Recipient)
	}

	for _, d := range deliveries {
		if _, err := tx.Exec(stmtsText[updateDelivery],
			sql.Named("id", d.id),
			sql.Named("dsn", rb.ParserInfos.DeliveryCode),
			sql.Named("status", parser.BouncedStatus),
		); err != nil {
			return errorutil.Wrap(err)
		}

		if _, err := tx.Exec(stmtsText[insertLogLineRef], d.id, tracking.ResultDeliveryLineRelayedBounce, rb.RecordTime.Unix(), rb.RecordSum); err != nil {
			return errorutil.Wrap(err)
		}
	}

	return nil
}
