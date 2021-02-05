// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package subcommand

import (
	"database/sql"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/migrator"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"path"
	"testing"
)

func init() {
	migrator.AddMigration("dummy", "1_dummy_table.go", UpTable, DownTable)
}

func UpTable(tx *sql.Tx) error {
	sql := `create table if not exists dummy(
		key string,
		value blob
	)`

	_, err := tx.Exec(sql)
	if err != nil {
		return errorutil.Wrap(err)
	}
	return nil
}

func DownTable(tx *sql.Tx) error {
	return nil
}

func init() {
	lmsqlite3.Initialize(lmsqlite3.Options{})
}

func TestDatabaseMigrationUp(t *testing.T) {
	Convey("Migration succeeds", t, func() {
		Convey("Run dummy migrations", func() {
			dir, clearDir := testutil.TempDir(t)
			defer clearDir()

			dbPath := path.Join(dir, "dummy.db")

			connPair, err := dbconn.Open(dbPath, 5)
			So(err, ShouldBeNil)

			err = migrator.Run(connPair.RwConn.DB, "dummy")
			So(err, ShouldBeNil)

			PerformMigrateDownTo(true, dir, "dummy", 0)
		})
	})
}
