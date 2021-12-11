// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package migrations

import (
	"database/sql"
	"github.com/rs/zerolog/log"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/migrator"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
)

const (
	// NOTE: we cannot import tracking due an import cycle :-(
	// so copying the values is needed!
	QueueMessageIDKey    = 12
	MessageIdFilenameKey = 43
	MessageIdLineKey     = 44
)

func init() {
	migrator.AddMigration("logtracker", "4_delete_messageids_table.go", upDeleteMessageIds, downDeleteMessageIds)
}

func migrateMessageIdTables(tx *sql.Tx) (err error) {
	//nolint:sqlclosecheck
	rows, err := tx.Query(`select id, messageid_id from queues where messageid_id is not null`)

	defer errorutil.UpdateErrorFromCloser(rows, &err)

	for rows.Next() {
		var (
			id        int64
			messageId int64
		)

		if err = rows.Scan(&id, &messageId); err != nil {
			return errorutil.Wrap(err)
		}

		if err = migrateMessageIdForQueue(tx, id, messageId); err != nil {
			return errorutil.Wrap(err)
		}
	}

	if err = rows.Err(); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func migrateMessageIdForQueue(tx *sql.Tx, queueId, messageId int64) (err error) {
	var (
		value    string
		filename string
		line     int64
	)

	if err = tx.QueryRow(`select value, filename, line from messageids where id = ?`, messageId).Scan(&value, &filename, &line); err != nil {
		return errorutil.Wrap(err)
	}

	log.Debug().Msgf("Migrating queue %v with messageid = %v on %v:%v", queueId, value, filename, line)

	if _, err = tx.Exec(`insert into queue_data(queue_id, key, value) values(?, ?, ?), (?, ?, ?), (?, ?, ?)`,
		queueId, QueueMessageIDKey, value,
		queueId, MessageIdFilenameKey, filename,
		queueId, MessageIdLineKey, line,
	); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

// TODO: remove the messageid_id column on queues!!!
func upDeleteMessageIds(tx *sql.Tx) error {
	if err := migrateMessageIdTables(tx); err != nil {
		return errorutil.Wrap(err)
	}

	sql := `
		drop table messageids;
		drop table processed_queues;
`

	if _, err := tx.Exec(sql); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func downDeleteMessageIds(tx *sql.Tx) error {
	// TODO: maybe implement it, restoring the messageids table?
	return nil
}
