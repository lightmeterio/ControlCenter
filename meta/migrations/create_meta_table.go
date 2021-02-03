// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0

package migrations

import (
	"database/sql"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/migrator"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
)

func init() {
	migrator.AddMigration("auth", "2_create_meta_table.go", UpMetaTable, DownMetaTable)
	migrator.AddMigration("master", "1_create_meta_table.go", UpMetaTable, DownMetaTable)
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
