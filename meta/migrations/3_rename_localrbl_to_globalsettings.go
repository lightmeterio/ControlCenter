// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package migrations

import (
	"database/sql"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/migrator"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
)

func init() {
	migrator.AddMigration("master", "3_rename_localrbl_to_globalsettings.go", upRenameSettings, downRenameSettings)
}

func upRenameSettings(tx *sql.Tx) error {
	_, err := tx.Exec("update meta set key = ? where key = ?", "global", "localrbl")
	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func downRenameSettings(tx *sql.Tx) error {
	return nil
}
