// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package postfixversion

import (
	"context"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/meta"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/postfixutil"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"strings"
	"testing"
	"time"
)

func init() {
	lmsqlite3.Initialize(lmsqlite3.Options{})
}

func getVersion() (*string, error) {
	var version *string
	err := meta.RetrieveJson(context.Background(), dbconn.Db("master"), SettingKey, &version)
	unwrappedErr := errorutil.TryToUnwrap(err)
	return version, unwrappedErr
}

func TestPostfixVersionPublisher(t *testing.T) {
	Convey("TestPostfixVersion", t, func() {
		_, closeDatabases := testutil.TempDatabases(t)
		defer closeDatabases()

		runner := meta.NewRunner(dbconn.Db("master"))
		done, cancel := runner.Run()

		defer func() {
			cancel()
			So(done(), ShouldBeNil)
		}()

		settingsWriter := runner.Writer()

		p := NewPublisher(settingsWriter)

		postfixutil.ReadFromTestReader(strings.NewReader("Mar 29 12:55:50 test1 postfix/postfix-script[15017]: starting the Postfix mail system"), p, 2020)
		time.Sleep(100 * time.Millisecond)

		Convey("Version unset", func() {
			version, err := getVersion()
			So(err, ShouldEqual, meta.ErrNoSuchKey)
			So(version, ShouldBeNil)
		})

		postfixutil.ReadFromTestReader(strings.NewReader("Mar 29 12:55:50 test1 postfix/master[15019]: daemon started -- version 3.4.14, configuration /etc/postfix"), p, 2020)
		time.Sleep(100 * time.Millisecond)

		Convey("Version set", func() {
			version, err := getVersion()
			So(err, ShouldBeNil)
			So(*version, ShouldEqual, "3.4.14")
		})

		postfixutil.ReadFromTestReader(strings.NewReader("Mar 29 12:55:50 test1 postfix/master[15019]: daemon started -- version 42.0, configuration /etc/postfix"), p, 2020)
		time.Sleep(100 * time.Millisecond)

		Convey("Version overriden", func() {
			version, err := getVersion()
			So(err, ShouldBeNil)
			So(*version, ShouldEqual, "42.0")
		})
	})
}
