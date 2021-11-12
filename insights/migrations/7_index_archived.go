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
	migrator.AddMigration("insights", "7_index_archived.go", upIndexStatus, downIndexStatus)
}

func upIndexStatus(tx *sql.Tx) error {
	sql := `create index insights_status_insight_id_index on insights_status(insight_id)`

	_, err := tx.Exec(sql)
	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func downIndexStatus(tx *sql.Tx) error {
	return nil
}
