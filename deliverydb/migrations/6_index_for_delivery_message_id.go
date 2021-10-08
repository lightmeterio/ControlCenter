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
	migrator.AddMigration("logs", "6_index_for_message_id.go", upAddMsgIndexIndex, downAddMsgIndexIndex)
}

func upAddMsgIndexIndex(tx *sql.Tx) error {
	// TODO: this is a temporary index to help fix issue #569 and will be removed on 2.0 when
	// we remove the messageids table!!!
	sql := `create index deliveries_messageid on deliveries(message_id, delivery_ts)`

	if _, err := tx.Exec(sql); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func downAddMsgIndexIndex(tx *sql.Tx) error {
	return nil
}
