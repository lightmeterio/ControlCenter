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
	migrator.AddMigration("logs", "7_log_line_ref.go", func(tx *sql.Tx) error {
		const sql = `
create table log_lines_ref(
	id integer primary key,
	delivery_id integer not null,
	ref_type integer not null,
	time integer not null,
	checksum integer not null
);

create index log_lines_ref_delivery_index on log_lines_ref(delivery_id, ref_type);
		`
		if _, err := tx.Exec(sql); err != nil {
			return errorutil.Wrap(err)
		}

		return nil

	}, func(*sql.Tx) error {
		return nil
	})
}
