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
	migrator.AddMigration("deliverydb", "4_add_queue_info.go", upAddQueueTables, downAddCreateDeliveryTables)
}

func upAddQueueTables(tx *sql.Tx) error {
	sql := `
		create table queues (
			id integer primary key,
			name text not null
		);

		create index queue_name_index on queues(name);

		create table delivery_queue (
			id integer primary key,
			queue_id int not null,
			delivery_id int not null
		);
	`

	if _, err := tx.Exec(sql); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func downAddCreateDeliveryTables(tx *sql.Tx) error {
	return nil
}
