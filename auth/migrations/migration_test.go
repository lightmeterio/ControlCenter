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
		Convey("Run migrations", func() {
			dir, clearDir := testutil.TempDir()
			defer clearDir()

			connPair, err := dbconn.NewConnPair(path.Join(dir, "auth.db"))
			So(err, ShouldBeNil)

			err = migrator.Run(connPair.RwConn.DB, "auth")
			So(err, ShouldBeNil)
		})
	})
}
