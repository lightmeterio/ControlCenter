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
	migrator.AddMigration("rawlogs", "1_create_logs_table.go", upCreateLogsTable, downCreateLogsTable)
}

func upCreateLogsTable(tx *sql.Tx) error {
	sql := `
	create table logs(
		id integer primary key,
		time integer not null,
		checksum integer not null,
		content text not null
	);
		
	-- this index is needed for searching for individual rows (by other subsystems)
	create index logs_sum_index on logs(time, checksum);
	
	-- this index is needed for the paginated search
	create index logs_query_index on logs(time, id, checksum);
`

	_, err := tx.Exec(sql)
	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func downCreateLogsTable(tx *sql.Tx) error {
	return errors.New(`Cannot migrate down from the first database schema`)
}
