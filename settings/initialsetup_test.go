package settings

import (
	"errors"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/meta"
	"gitlab.com/lightmeter/controlcenter/util"
	"io/ioutil"
	"os"
	"path"
	"testing"
)

func tempDir() string {
	dir, e := ioutil.TempDir("", "lightmeter-tests-*")
	if e != nil {
		panic("error creating temp dir")
	}
	return dir
}

type fakeNewsletterSubscriber struct {
	shouldFailToSubscribe bool
	hasSubscribed         bool
}

func (s *fakeNewsletterSubscriber) Subscribe(email string) error {
	if s.shouldFailToSubscribe {
		return errors.New(`Fail to Subscribe!!!`)
	}

	s.hasSubscribed = true
	return nil
}

func TestInitialSetup(t *testing.T) {
	Convey("Initial Setup", t, func() {
		dir := tempDir()
		defer os.RemoveAll(dir)

		conn, err := dbconn.NewConnPair(path.Join(dir, "master.db"))
		So(err, ShouldBeNil)
		defer func() { util.MustSucceed(conn.Close(), "") }()

		meta, err := meta.NewMetaDataHandler(conn)
		So(err, ShouldBeNil)
		defer func() { util.MustSucceed(meta.Close(), "") }()

		newsletterSubscriber := &fakeNewsletterSubscriber{}

		m, err := NewMasterConf(meta, newsletterSubscriber)
		So(err, ShouldBeNil)
		defer func() { util.MustSucceed(m.Close(), "") }()

		Convey("Invalid Mail Kind", func() {
			So(errors.Is(m.SetInitialOptions(InitialSetupOptions{
				SubscribeToNewsletter: true,
				MailKind:              "Lalala"},
			), ErrInvalidInintialSetupOption), ShouldBeTrue)
		})

		Convey("Fails to Subscribe", func() {
			newsletterSubscriber.shouldFailToSubscribe = true

			So(errors.Is(m.SetInitialOptions(InitialSetupOptions{
				SubscribeToNewsletter: true,
				MailKind:              MailKindMarketing,
				Email:                 "user@example.com"},
			), ErrFailedToSubscribeToNewsletter), ShouldBeTrue)
		})

		Convey("Succeeds subscribing", func() {
			err := m.SetInitialOptions(InitialSetupOptions{
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
			err := m.SetInitialOptions(InitialSetupOptions{
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
