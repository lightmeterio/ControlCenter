// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package migrations

import (
	"database/sql"
	"errors"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/migrator"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
)

func init() {
	migrator.AddMigration("connections", "2_add_support_for_imap.go", upAddProtocolColumn, downAddProtocolColumn)
}

func upAddProtocolColumn(tx *sql.Tx) error {
	// Before this migration, we supported only smtp connections
	// so we set all the previous connections with the `connectionstats.ProtocolSMTP = 0` flag (column default).
	sql := `
	alter table connections add column protocol integer not null default 0;
	`

	_, err := tx.Exec(sql)
	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func downAddProtocolColumn(tx *sql.Tx) error {
	return errors.New(`Cannot migrate down from the first database schema`)
}
