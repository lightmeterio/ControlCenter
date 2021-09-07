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
	migrator.AddMigration("auth", "2_create_meta_table.go", UpMetaTable, DownMetaTable)
	migrator.AddMigration("master", "1_create_meta_table.go", UpMetaTable, DownMetaTable)
	migrator.AddMigration("intel-collector", "3_create_meta_table.go", UpMetaTable, DownMetaTable)

	migrator.AddMigration("auth", "4_index_meta_table.go", IndexMetaTable, DropIndexMetaTable)
	migrator.AddMigration("master", "5_index_meta_table.go", IndexMetaTable, DropIndexMetaTable)
	migrator.AddMigration("intel-collector", "6_index_meta_table.go", IndexMetaTable, DropIndexMetaTable)
}

func UpMetaTable(tx *sql.Tx) error {
	sql := `create table if not exists meta(
		key string,
		value blob
	)`

	_, err := tx.Exec(sql)
	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func DownMetaTable(tx *sql.Tx) error {
	return nil
}

func IndexMetaTable(tx *sql.Tx) error {
	sql := `create unique index if not exists unique_key on meta (key)`

	_, err := tx.Exec(sql)
	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func DropIndexMetaTable(tx *sql.Tx) error {
	return nil
}
