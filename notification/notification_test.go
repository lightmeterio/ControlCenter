package notification

import (
	"context"
	"fmt"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/meta"
	"gitlab.com/lightmeter/controlcenter/settings"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"sync/atomic"
	"testing"
)

var (
	dummyContext = context.Background()
)

func init() {
	lmsqlite3.Initialize(lmsqlite3.Options{})
}

type fakeContent struct{}

func (c *fakeContent) String() string {
	return "Hell world!, Mister Donutloop"
}

type dummySubscriber struct{}

func (*dummySubscriber) Subscribe(context context.Context, email string) error {
	return nil
}

func newSubscriber() *dummySubscriber {
	return &dummySubscriber{}
}

func TestSendNotification(t *testing.T) {

	Convey("Notification", t, func() {
		conn, closeConn := testutil.TempDBConnection()
		defer closeConn()

		m, err := meta.NewHandler(conn, "master")
		So(err, ShouldBeNil)

		defer func() { errorutil.MustSucceed(m.Close()) }()

		master, err := settings.NewMasterConf(m, newSubscriber())
		So(err, ShouldBeNil)

		slackSettings := settings.SlackNotificationsSettings{
			Channel:     "general",
			Kind:        "slack",
			BearerToken: "xoxb-1388191062644-1385067635637-5dvVTcz77UHTyFDwmjZY6sEz",
			Enabled:     true,
		}

		err = master.SetSlackNotificationsSettings(dummyContext, slackSettings)
		So(err, ShouldBeNil)

		center := New(master)
		So(err, ShouldBeNil)

		content := new(fakeContent)
		notification := Notification{
			ID:      0,
			Content: content,
		}

		Convey("Success", func() {
			Convey("Do subscribe", func() {
				err := center.Notify(notification)
				So(err, ShouldBeNil)
			})
		})
	})
}

func TestSendNotificationMissingConf(t *testing.T) {

	Convey("Notification", t, func() {
		conn, closeConn := testutil.TempDBConnection()
		defer closeConn()

		m, err := meta.NewHandler(conn, "master")
		So(err, ShouldBeNil)

		defer func() { errorutil.MustSucceed(m.Close()) }()

		master, err := settings.NewMasterConf(m, newSubscriber())
		So(err, ShouldBeNil)

		center := New(master)
		So(err, ShouldBeNil)

		content := new(fakeContent)
		notification := Notification{
			ID:      0,
			Content: content,
		}

		Convey("Success", func() {
			Convey("Do subscribe", func() {
				err := center.Notify(notification)
				So(err, ShouldBeNil)
			})
		})
	})
}

type fakeapi struct {
	Counter int32
}

func (s *fakeapi) PostMessage(stringer fmt.Stringer) error {
	fmt.Println(stringer)
	atomic.AddInt32(&s.Counter, 1)
	return nil
}

func TestFakeSendNotification(t *testing.T) {

	Convey("Notification", t, func() {
		conn, closeConn := testutil.TempDBConnection()
		defer closeConn()

		m, err := meta.NewHandler(conn, "master")
		So(err, ShouldBeNil)

		defer func() { errorutil.MustSucceed(m.Close()) }()

		master, err := settings.NewMasterConf(m, newSubscriber())
		So(err, ShouldBeNil)

		slackSettings := settings.SlackNotificationsSettings{
			Channel:     "general",
			Kind:        "slack",
			BearerToken: "xoxb-1388191062644-1385067635637-5dvVTcz77UHTyFDwmjZY6sEz",
			Enabled:     true,
		}

		err = master.SetSlackNotificationsSettings(dummyContext, slackSettings)
		So(err, ShouldBeNil)

		fakeapi := &fakeapi{}

		centerInterface := New(master)
		c := centerInterface.(*center)
		c.slackapi = fakeapi

		content := new(fakeContent)
		Notification := Notification{
			ID:      0,
			Content: content,
		}

		Convey("Success", func() {
			Convey("Do subscribe", func() {
				err := c.Notify(Notification)
				So(err, ShouldBeNil)
				So(fakeapi.Counter, ShouldEqual, 1)
			})
		})
	})
}

func TestFakeSendNotificationDisabled(t *testing.T) {

	Convey("Notification", t, func() {
		conn, closeConn := testutil.TempDBConnection()
		defer closeConn()

		m, err := meta.NewHandler(conn, "master")
		So(err, ShouldBeNil)

		defer func() { errorutil.MustSucceed(m.Close()) }()

		master, err := settings.NewMasterConf(m, newSubscriber())
		So(err, ShouldBeNil)

		slackSettings := settings.SlackNotificationsSettings{
			Channel:     "general",
			Kind:        "slack",
			BearerToken: "xoxb-1388191062644-1385067635637-5dvVTcz77UHTyFDwmjZY6sEz",
			Enabled:     false,
		}

		err = master.SetSlackNotificationsSettings(dummyContext, slackSettings)
		So(err, ShouldBeNil)

		fakeapi := &fakeapi{}

		centerInterface := New(master)
		c := centerInterface.(*center)
		c.slackapi = fakeapi

		content := new(fakeContent)
		Notification := Notification{
			ID:      0,
			Content: content,
		}

		Convey("Success", func() {
			Convey("Do subscribe", func() {
				err := c.Notify(Notification)
				So(err, ShouldBeNil)
				So(fakeapi.Counter, ShouldEqual, 0)
			})
		})
	})
}
