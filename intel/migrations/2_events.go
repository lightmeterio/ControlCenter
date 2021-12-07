// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package collector

import (
	"database/sql"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/migrator"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
)

func init() {
	migrator.AddMigration("intel-collector", "2_events.go", upCreateEvent, downCreateEvent)
}

func upCreateEvent(tx *sql.Tx) error {
	sql := `create table if not exists events(
			id integer primary key,
			received_time integer not null,
			event_id text not null,
			content_type blob not null,
			content text not null,
			dismissing_time integer
		);

		create index events_content_type_index on events(content_type);

		create table current_blocked_ips(
			id integer primary key
		);
		`

	_, err := tx.Exec(sql)
	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func downCreateEvent(tx *sql.Tx) error {
	return nil
}
