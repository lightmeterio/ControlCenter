// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package migrations

import (
	"database/sql"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/migrator"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
)

func init() {
	migrator.AddMigration("logtracker", "3_wipe_tracking_data.go", upWipeData, downWipeData)
}

func upWipeData(tx *sql.Tx) error {
	// Yes, delete data from all tables, as the old data was inconsistent.
	sql := `
		delete from queues;
		delete from results;
		delete from result_data;
		delete from messageids;
		delete from queue_parenting;
		delete from queue_data;
		delete from connections;
		delete from connection_data;
		delete from pids;
		delete from notification_queues;
		delete from processed_queues;
`

	if _, err := tx.Exec(sql); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func downWipeData(tx *sql.Tx) error {
	return nil
}
