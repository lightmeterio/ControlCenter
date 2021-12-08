// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package migrations

import (
	"database/sql"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/migrator"
	metaMigrations "gitlab.com/lightmeter/controlcenter/metadata/migrations"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
)

func init() {
	migrator.AddMigration("insights", "3_metadata.go", metaMigrations.UpMetaTable, metaMigrations.DownMetaTable)
	migrator.AddMigration("insights", "4_historical_import.go", upStatus, downStatus)
	migrator.AddMigration("insights", "6_index_meta_table.go", metaMigrations.IndexMetaTable, metaMigrations.DropIndexMetaTable)
}

func upStatus(tx *sql.Tx) error {
	sql := `
		create table insights_status(
			id integer primary key,
			insight_id integer not null,
			status integer not null,
			timestamp integer not null
		);

		create index insights_status_status_index on insights_status(timestamp, status); 

		create table import_progress(
			id integer primary key,
			value integer,
			timestamp integer,
			exec_timestamp integer not null
		);
`

	_, err := tx.Exec(sql)
	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func downStatus(tx *sql.Tx) error {
	_, err := tx.Exec(`drop table insights_status; drop table import_progress`)
	return err
}
