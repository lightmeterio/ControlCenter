// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0

package subcommand

import (
	"context"
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
			dir, clearDir := testutil.TempDir(t)
			defer clearDir()

			auth, err := auth.NewAuth(dir, auth.Options{})
			ShouldBeNil(err)

			email := "marcel@lightmeter.com"

			_, err = auth.Register(context.Background(), email, "donutloop", "l;sdkfl;s;ldfkkl")
			ShouldBeNil(err)

			PerformPasswordReset(true, dir, email, "kshjdkljdfklsjfljsdjkf")
		})
	})
}
