// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package settings

import (
	"context"
	"errors"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/meta"
	"gitlab.com/lightmeter/controlcenter/notification/slack"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
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

type fakeNewsletterSubscriber struct {
	shouldFailToSubscribe bool
	hasSubscribed         bool
}

func (s *fakeNewsletterSubscriber) Subscribe(context context.Context, email string) error {
	if s.shouldFailToSubscribe {
		return errors.New(`Fail to Subscribe!!!`)
	}

	s.hasSubscribed = true
	return nil
}

func TestMessengerSettings(t *testing.T) {
	Convey("messenger settings", t, func() {
		context, _ := context.WithTimeout(context.Background(), 500*time.Millisecond)

		conn, closeConn := testutil.TempDBConnection(t)
		defer closeConn()

		m, err := meta.NewHandler(conn, "master")
		So(err, ShouldBeNil)

		defer func() { errorutil.MustSucceed(m.Close()) }()

		runner := meta.NewRunner(m)
		writer := runner.Writer()
		done, cancel := runner.Run()

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

		conn, closeConn := testutil.TempDBConnection(t)
		defer closeConn()

		m, err := meta.NewHandler(conn, "master")
		So(err, ShouldBeNil)
		defer func() { errorutil.MustSucceed(m.Close()) }()

		runner := meta.NewRunner(m)
		writer := runner.Writer()
		done, cancel := runner.Run()

		defer func() { cancel(); done() }()

		newsletterSubscriber := &fakeNewsletterSubscriber{}

		s := NewInitialSetupSettings(newsletterSubscriber)

		Convey("Invalid Mail Kind", func() {
			So(errors.Is(s.Set(context, writer, InitialOptions{
				SubscribeToNewsletter: true,
				MailKind:              "Lalala"},
			), ErrInvalidMailKindOption), ShouldBeTrue)
		})

		Convey("Fails to Subscribe", func() {
			newsletterSubscriber.shouldFailToSubscribe = true

			So(errors.Is(s.Set(context, writer, InitialOptions{
				SubscribeToNewsletter: true,
				MailKind:              MailKindMarketing,
				Email:                 "user@example.com"},
			), ErrFailedToSubscribeToNewsletter), ShouldBeTrue)
		})

		Convey("Succeeds subscribing", func() {
			err := s.Set(context, writer, InitialOptions{
				SubscribeToNewsletter: true,
				MailKind:              MailKindMarketing,
				Email:                 "user@example.com"},
			)

			So(err, ShouldBeNil)
			So(newsletterSubscriber.hasSubscribed, ShouldBeTrue)

			r, err := m.Reader.Retrieve(dummyContext, "mail_kind")
			So(err, ShouldBeNil)
			So(r, ShouldEqual, MailKindMarketing)

			r, err = m.Reader.Retrieve(dummyContext, "subscribe_newsletter")
			So(err, ShouldBeNil)
			So(r, ShouldEqual, 1)
		})

		Convey("Succeeds not subscribing", func() {
			err := s.Set(context, writer, InitialOptions{
				SubscribeToNewsletter: false,
				MailKind:              MailKindTransactional,
				Email:                 "user@example.com"},
			)

			So(err, ShouldBeNil)
			So(newsletterSubscriber.hasSubscribed, ShouldBeFalse)

			r, err := m.Reader.Retrieve(dummyContext, "mail_kind")
			So(err, ShouldBeNil)
			So(r, ShouldEqual, MailKindTransactional)

			r, err = m.Reader.Retrieve(dummyContext, "subscribe_newsletter")
			So(err, ShouldBeNil)
			So(r, ShouldEqual, 0)
		})
	})
}
