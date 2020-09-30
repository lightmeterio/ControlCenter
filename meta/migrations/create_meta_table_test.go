package migrations

import (
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	_ "gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/migrator"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"path"
	"testing"
)

func init() {
	lmsqlite3.Initialize(lmsqlite3.Options{})
}

func TestDatabaseMigrationUp(t *testing.T) {
	Convey("Migration succeeds", t, func() {
		dir, clearDir := testutil.TempDir()
		defer clearDir()

		Convey("Run auth migrations", func() {
			connPair, err := dbconn.NewConnPair(path.Join(dir, "auth.db"))
			So(err, ShouldBeNil)

			err = migrator.Run(connPair.RwConn.DB, "auth")
			So(err, ShouldBeNil)
		})

		Convey("Run master migrations", func() {
			connPair, err := dbconn.NewConnPair(path.Join(dir, "master.db"))
			So(err, ShouldBeNil)

			err = migrator.Run(connPair.RwConn.DB, "master")
			So(err, ShouldBeNil)
		})
	})
}
