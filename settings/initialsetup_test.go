package settings

import (
	"context"
	"errors"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/meta"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"os"
	"path"
	"testing"
	"time"
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

func TestInitialSetup(t *testing.T) {
	Convey("Initial Setup", t, func() {
		context, _ := context.WithTimeout(context.Background(), 500*time.Millisecond)

		dir := testutil.TempDir()
		defer os.RemoveAll(dir)

		conn, err := dbconn.NewConnPair(path.Join(dir, "master.db"))
		So(err, ShouldBeNil)
		defer func() { errorutil.MustSucceed(conn.Close(), "") }()

		meta, err := meta.NewMetaDataHandler(conn)
		So(err, ShouldBeNil)
		defer func() { errorutil.MustSucceed(meta.Close(), "") }()

		newsletterSubscriber := &fakeNewsletterSubscriber{}

		m, err := NewMasterConf(meta, newsletterSubscriber)
		So(err, ShouldBeNil)
		defer func() { errorutil.MustSucceed(m.Close(), "") }()

		Convey("Invalid Mail Kind", func() {
			So(errors.Is(m.SetInitialOptions(context, InitialSetupOptions{
				SubscribeToNewsletter: true,
				MailKind:              "Lalala"},
			), ErrInvalidMailKindOption), ShouldBeTrue)
		})

		Convey("Fails to Subscribe", func() {
			newsletterSubscriber.shouldFailToSubscribe = true

			So(errors.Is(m.SetInitialOptions(context, InitialSetupOptions{
				SubscribeToNewsletter: true,
				MailKind:              MailKindMarketing,
				Email:                 "user@example.com"},
			), ErrFailedToSubscribeToNewsletter), ShouldBeTrue)
		})

		Convey("Succeeds subscribing", func() {
			err := m.SetInitialOptions(context, InitialSetupOptions{
				SubscribeToNewsletter: true,
				MailKind:              MailKindMarketing,
				Email:                 "user@example.com"},
			)

			So(err, ShouldBeNil)
			So(newsletterSubscriber.hasSubscribed, ShouldBeTrue)

			r, err := meta.Retrieve("mail_kind")
			So(err, ShouldBeNil)
			So(len(r), ShouldEqual, 1)
			So(r[0], ShouldEqual, MailKindMarketing)

			r, err = meta.Retrieve("subscribe_newsletter")
			So(err, ShouldBeNil)
			So(len(r), ShouldEqual, 1)
			So(r[0], ShouldEqual, 1)
		})

		Convey("Succeeds not subscribing", func() {
			err := m.SetInitialOptions(context, InitialSetupOptions{
				SubscribeToNewsletter: false,
				MailKind:              MailKindTransactional,
				Email:                 "user@example.com"},
			)

			So(err, ShouldBeNil)
			So(newsletterSubscriber.hasSubscribed, ShouldBeFalse)

			r, err := meta.Retrieve("mail_kind")
			So(err, ShouldBeNil)
			So(len(r), ShouldEqual, 1)
			So(r[0], ShouldEqual, MailKindTransactional)

			r, err = meta.Retrieve("subscribe_newsletter")
			So(err, ShouldBeNil)
			So(len(r), ShouldEqual, 1)
			So(r[0], ShouldEqual, 0)
		})
	})
}
