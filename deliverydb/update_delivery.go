// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package deliverydb

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/pkg/dbrunner"
	"gitlab.com/lightmeter/controlcenter/pkg/postfix"
	parser "gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser"
	"gitlab.com/lightmeter/controlcenter/tracking"
	"gitlab.com/lightmeter/controlcenter/util/emailutil"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
)

type delivery struct {
	id, ts int64
}

func updateDeliveryWithBounceInfoAction(initialTime time.Time, actions chan dbrunner.Action, r postfix.Record, ttl int) dbrunner.Action {
	return func(tx *sql.Tx, stmts dbconn.TxPreparedStmts) (err error) {
		lrb, ok := r.Payload.(parser.LightmeterRelayedBounce)
		if !ok {
			log.Fatal().Msg("Crash now!!!!!")
			return errorutil.Wrap(fmt.Errorf("Can't cast object into parser.LightmeterRelayedBounce: %v", r.Payload))
		}

		if ttl < 0 {
			log.Warn().Msgf("Could not find a delivery to update between %v (%v) and %v (%v) for relayed bounce %#v", initialTime, initialTime.Unix(), r.Time, r.Time.Unix(), lrb)

			return nil
		}

		updated, err := updateDeliveryWithBounceInfo(initialTime, tx, r, lrb)
		if err != nil {
			return err
		}

		// NOTE: updateDeliveryWithBounceInfo can wait a few seconds for the delivery to be created -- don't block the main thread
		if !updated {
			go func() {
				time.Sleep(1 * time.Second)
				log.Debug().Msgf("Retry action after one second")
				actions <- updateDeliveryWithBounceInfoAction(initialTime, actions, r, ttl-1)
			}()
		}

		return nil
	}
}

func updateDeliveryWithBounceInfo(initialTime time.Time, tx *sql.Tx, r postfix.Record, p parser.LightmeterRelayedBounce) (updated bool, err error) {
	senderU, senderD, err := emailutil.Split(p.Sender)
	if err != nil {
		return false, errorutil.Wrap(err)
	}

	recipientU, recipientD, err := emailutil.Split(p.Recipient)
	if err != nil {
		return false, errorutil.Wrap(err)
	}

	rows, err := tx.Query(stmtsText[selectDeliveries],
		sql.Named("sender_user", senderU),
		sql.Named("sender_domain", senderD),
		sql.Named("recipient_user", recipientU),
		sql.Named("recipient_domain", recipientD),
		sql.Named("an_hour_ago", initialTime.Unix()),
		sql.Named("current_time", r.Time.Unix()),
	)

	if err != nil {
		return false, errorutil.Wrap(err)
	}

	defer errorutil.UpdateErrorFromCloser(rows, &err)

	deliveries := []delivery{}

	for rows.Next() {
		var d delivery

		if err := rows.Scan(&d.id, &d.ts); err != nil {
			return false, errorutil.Wrap(err)
		}

		deliveries = append(deliveries, d)
	}

	if err := rows.Err(); err != nil {
		return false, errorutil.Wrap(err)
	}

	if len(deliveries) == 0 {
		log.Debug().Msgf("No relayed bounces were updated between %v and %v. Try again later?! record = %#v", initialTime, r.Time, r)
		return false, nil
	}

	if len(deliveries) > 1 {
		log.Warn().Msgf("Relayed bounce updates %d lines (%s => %s)", len(deliveries), p.Sender, p.Recipient)
	}

	for _, d := range deliveries {
		if _, err := tx.Exec(stmtsText[updateDelivery],
			sql.Named("id", d.id),
			sql.Named("dsn", p.DeliveryCode),
			sql.Named("status", parser.BouncedStatus),
		); err != nil {
			return false, errorutil.Wrap(err)
		}

		if _, err := tx.Exec(stmtsText[insertLogLineRef], d.id, tracking.ResultDeliveryLineRelayedBounce, r.Time.Unix(), r.Sum); err != nil {
			return false, errorutil.Wrap(err)
		}
	}

	return len(deliveries) > 0, nil
}
