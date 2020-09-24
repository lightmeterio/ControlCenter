package subcommand

import (
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/auth"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"testing"
)

func init() {
	lmsqlite3.Initialize(lmsqlite3.Options{})
}

func TestDatabaseRegisterUsername(t *testing.T) {

	Convey("Password reset", t, func() {
		Convey("Do password reset", func() {
			workspace := testutil.TempDir()

			auth, err := auth.NewAuth(workspace, auth.Options{})
			ShouldBeNil(err)

			email := "marcel@lightmeter.com"

			err = auth.Register(email, "donutloop", "l;sdkfl;s;ldfkkl")
			ShouldBeNil(err)

			PerformPasswordReset(true, workspace, email, "kshjdkljdfklsjfljsdjkf")
		})
	})
}
