// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package settings

import (
	"context"
	"errors"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/metadata"
	"gitlab.com/lightmeter/controlcenter/newsletter"
	"gitlab.com/lightmeter/controlcenter/notification/slack"
	"gitlab.com/lightmeter/controlcenter/pkg/runner"
	"gitlab.com/lightmeter/controlcenter/util/stringutil"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"testing"
	"time"
)

var (
	dummyContext = context.Background()
)

func init() {
	lmsqlite3.Initialize(lmsqlite3.Options{})
}

func TestMessengerSettings(t *testing.T) {
	Convey("messenger settings", t, func() {
		context, _ := context.WithTimeout(context.Background(), 500*time.Millisecond)

		conn, closeConn := testutil.TempDBConnectionMigrated(t, "master")
		defer closeConn()

		m, err := metadata.NewHandler(conn)
		So(err, ShouldBeNil)

		writeRunner := metadata.NewSerialWriteRunner(m)
		writer := writeRunner.Writer()
		done, cancel := runner.Run(writeRunner)

		defer func() { cancel(); done() }()

		Convey("valid messenger settings", func() {
			s := slack.Settings{
				Channel:     "donutloop",
				BearerToken: stringutil.MakeSensitive("fjslfjjsdfljlskjfkdjs"),
			}

			err := slack.SetSettings(context, writer, s)
			So(err, ShouldBeNil)

			retrievedSetting, err := slack.GetSettings(dummyContext, m.Reader)
			So(err, ShouldBeNil)

			So(retrievedSetting, ShouldResemble, &s)
		})
	})
}

func TestInitialSetup(t *testing.T) {
	Convey("Test Initial Setup", t, func() {
		context, _ := context.WithTimeout(context.Background(), 500*time.Millisecond)

		conn, closeConn := testutil.TempDBConnectionMigrated(t, "master")
		defer closeConn()

		m, err := metadata.NewHandler(conn)
		So(err, ShouldBeNil)

		writeRunner := metadata.NewSerialWriteRunner(m)
		writer := writeRunner.Writer()
		done, cancel := runner.Run(writeRunner)

		defer func() { cancel(); done() }()

		newsletterSubscriber := &newsletter.FakeNewsletterSubscriber{}

		s := NewInitialSetupSettings(newsletterSubscriber)

		Convey("Fails to Subscribe", func() {
			newsletterSubscriber.ShouldFailToSubscribe = true

			So(errors.Is(s.Set(context, writer, InitialOptions{
				SubscribeToNewsletter: true,
				Email:                 "user@example.com"},
			), ErrFailedToSubscribeToNewsletter), ShouldBeTrue)
		})

		Convey("Succeeds subscribing", func() {
			err := s.Set(context, writer, InitialOptions{
				SubscribeToNewsletter: true,
				Email:                 "user@example.com"},
			)

			So(err, ShouldBeNil)
			So(newsletterSubscriber.HasSubscribed, ShouldBeTrue)

			r, err := m.Reader.Retrieve(dummyContext, "subscribe_newsletter")
			So(err, ShouldBeNil)
			So(r, ShouldEqual, 1)
		})

		Convey("Succeeds not subscribing", func() {
			err := s.Set(context, writer, InitialOptions{
				SubscribeToNewsletter: false,
				Email:                 "user@example.com"},
			)

			So(err, ShouldBeNil)
			So(newsletterSubscriber.HasSubscribed, ShouldBeFalse)

			r, err := m.Reader.Retrieve(dummyContext, "subscribe_newsletter")
			So(err, ShouldBeNil)
			So(r, ShouldEqual, 0)
		})
	})
}
