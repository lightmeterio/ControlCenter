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
	migrator.AddMigration("logtracker", "5_prenotification_table.go", func(tx *sql.Tx) error {
		if _, err := tx.Exec(`create table prenotification_results(
			id integer primary key,
			result_id integer not null,
			queue_id integer not null
		);

		create index prenotification_queue_id_index on prenotification_results(queue_id);
		`); err != nil {
			return errorutil.Wrap(err)
		}

		return nil

	}, func(*sql.Tx) error {
		return nil
	})
}
