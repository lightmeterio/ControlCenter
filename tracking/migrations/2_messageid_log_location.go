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
	migrator.AddMigration("logtracker", "2_messageid_log_location.go", upUpdateMessageIdTable, downUpdateMessageIdTable)
}

func upUpdateMessageIdTable(tx *sql.Tx) error {
	sql := `
	alter table messageids add column filename text;
	alter table messageids add column line integer
`

	if _, err := tx.Exec(sql); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func downUpdateMessageIdTable(tx *sql.Tx) error {
	return nil
}
