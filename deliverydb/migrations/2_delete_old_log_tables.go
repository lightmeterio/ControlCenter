// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0

package migrations

import (
	"database/sql"
	"errors"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/migrator"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
)

func init() {
	migrator.AddMigration("deliverydb", "2_delete_old_log_tables.go", upDeleteOldTables, downDeleteOldTables)
}

func upDeleteOldTables(tx *sql.Tx) error {
	sql := `
	drop index postfix_smtp_message_time_index;
	drop table postfix_smtp_message_status;
`

	_, err := tx.Exec(sql)
	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func downDeleteOldTables(tx *sql.Tx) error {
	return errors.New(`Cannot migrate down to the old database tables`)
}
