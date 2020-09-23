package subcommand

import (
	"database/sql"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/migrator"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	. "github.com/smartystreets/goconvey/convey"
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
			workspace := testutil.TempDir()

			dbPath := path.Join(workspace, "dummy.db")

			connPair, err := dbconn.NewConnPair(dbPath)
			So(err, ShouldBeNil)

			err = migrator.Run(connPair.RwConn.DB, "dummy")
			So(err, ShouldBeNil)

			PerformMigrateDownTo(true, workspace, "dummy", 0)
		})
	})
}