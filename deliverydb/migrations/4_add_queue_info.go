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

		create index delivery_queue_queue_index on delivery_queue(queue_id);
		create index delivery_queue_delivery_index on delivery_queue(delivery_id);

		create table queue_parenting (
			id integer primary key,
			parent_queue_id int not null,
			child_queue_id int not null,
			type int not null
		);

		create index queue_parenting_parent_index on queue_parenting(parent_queue_id);
		create index queue_parenting_child_index on queue_parenting(child_queue_id);
	`

	if _, err := tx.Exec(sql); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func downAddCreateDeliveryTables(tx *sql.Tx) error {
	return nil
}
