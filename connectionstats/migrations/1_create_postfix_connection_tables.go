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
	migrator.AddMigration("connectionstats", "1_create_postfix_connection_tables.go", upCreateTables, downCreateTables)
}

func upCreateTables(tx *sql.Tx) error {
	sql := `
	create table connections(
		id integer primary key,
		disconnection_ts integer not null,
		ip blob not null
	);

	create table commands(
		id integer primary key,
		connection_id integer not null,
		cmd integer not null,
		success integer not null,
		total integer not null
	);
		
	create index connection_disconnection_time_index on connections(disconnection_ts);
	create index connection_ip_index on connections(ip);
	create index commands_connection_id_index on commands(connection_id);
`

	_, err := tx.Exec(sql)
	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func downCreateTables(tx *sql.Tx) error {
	return errors.New(`Cannot migrate down from the first database schema`)
}
