// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package postfixversion

import (
	"context"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/metadata"
	"gitlab.com/lightmeter/controlcenter/pkg/runner"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/postfixutil"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"strings"
	"testing"
	"time"
)

func init() {
	lmsqlite3.Initialize(lmsqlite3.Options{})
}

func getVersion(settingsReader metadata.Reader) (*string, error) {
	var version *string
	err := settingsReader.RetrieveJson(context.Background(), SettingsKey, &version)
	unwrappedErr := errorutil.TryToUnwrap(err)
	return version, unwrappedErr
}

func TestPostfixVersionPublisher(t *testing.T) {
	Convey("TestPostfixVersion", t, func() {
		settingdDB, removeDB := testutil.TempDBConnectionMigrated(t, "master")
		defer removeDB()

		handler, err := metadata.NewHandler(settingdDB)
		So(err, ShouldBeNil)

		writeRunner := metadata.NewSerialWriteRunner(handler)
		done, cancel := runner.Run(writeRunner)

		defer func() {
			cancel()
			So(done(), ShouldBeNil)
		}()

		settingsWriter := writeRunner.Writer()
		settingsReader := handler.Reader

		clock := &timeutil.FakeClock{Time: timeutil.MustParseTime(`2020-08-10 10:00:00 +0000`)}

		p := NewPublisher(settingsWriter)

		postfixutil.ReadFromTestReader(strings.NewReader("Mar 29 12:55:50 test1 postfix/postfix-script[15017]: starting the Postfix mail system"), p, 2020, clock)
		time.Sleep(100 * time.Millisecond)

		Convey("Version unset", func() {
			version, err := getVersion(settingsReader)
			So(err, ShouldEqual, metadata.ErrNoSuchKey)
			So(version, ShouldBeNil)
		})

		postfixutil.ReadFromTestReader(strings.NewReader("Mar 29 12:55:50 test1 postfix/master[15019]: daemon started -- version 3.4.14, configuration /etc/postfix"), p, 2020, clock)
		time.Sleep(100 * time.Millisecond)

		Convey("Version set", func() {
			version, err := getVersion(settingsReader)
			So(err, ShouldBeNil)
			So(*version, ShouldEqual, "3.4.14")
		})

		postfixutil.ReadFromTestReader(strings.NewReader("Mar 29 12:55:50 test1 postfix/master[15019]: daemon started -- version 42.0, configuration /etc/postfix"), p, 2019, clock)
		time.Sleep(100 * time.Millisecond)

		Convey("Version overriden", func() {
			version, err := getVersion(settingsReader)
			So(err, ShouldBeNil)
			So(*version, ShouldEqual, "42.0")
		})
	})
}
