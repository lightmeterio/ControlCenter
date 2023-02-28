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
	migrator.AddMigration("logs", "8_msgid_replied.go", func(tx *sql.Tx) error {
		const sql = `
create table messageids_replies(
	id integer primary key,
	original_id integer not null,
	reply_id integer not null
);

create index messageids_replies_original_index on messageids_replies(original_id);

create index messageids_replies_reply_index on messageids_replies(reply_id);
`
		if _, err := tx.Exec(sql); err != nil {
			return errorutil.Wrap(err)
		}

		return nil

	}, func(*sql.Tx) error {
		return nil
	})
}
