// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package collector

import (
	"database/sql"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/migrator"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"

	// The meta table is defined in the meta package
	_ "gitlab.com/lightmeter/controlcenter/metadata/migrations"
)

func init() {
	migrator.AddMigration("intel-collector", "1_basic_intel.go", upBasic, downBasic)
}

func upBasic(tx *sql.Tx) error {
	sql := `create table if not exists queued_reports(
			id integer primary key,
			time integer not null,
			dispatched_time integer not null,
			identifier text not null,
			value blob not null
		);

		create index queued_reports_dispatch_time on queued_reports(dispatched_time);

		create table dispatch_times(
			id integer primary key,
			time integer not null
		);
	`

	_, err := tx.Exec(sql)
	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func downBasic(tx *sql.Tx) error {
	return nil
}
