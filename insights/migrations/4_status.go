// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package migrations

import (
	"database/sql"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/migrator"
	metaMigrations "gitlab.com/lightmeter/controlcenter/meta/migrations"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
)

func init() {
	migrator.AddMigration("insights", "3_meta.go", metaMigrations.UpMetaTable, metaMigrations.DownMetaTable)
	migrator.AddMigration("insights", "4_status.go", upStatus, downStatus)
}

func upStatus(tx *sql.Tx) error {
	sql := `create table if not exists insights_status(
			id integer primary key,
			insight_id integer not null,
			status integer not null,
			timestamp integet not null
		);

		create index if not exists insights_status_status_index on insights_status(timestamp, status); 
`

	_, err := tx.Exec(sql)
	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func downStatus(tx *sql.Tx) error {
	return nil
}
