package tracking

import (
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/data"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/logeater/filelogsource"
	"gitlab.com/lightmeter/controlcenter/logeater/logsource"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"os"
	"testing"
	"time"
)

func init() {
	lmsqlite3.Initialize(lmsqlite3.Options{})
}

func openFile(name string) *os.File {
	f, err := os.Open(name)
	errorutil.MustSucceed(err)
	return f
}

type fakeResultPublisher struct {
	results []Result
}

func (p *fakeResultPublisher) Publish(r Result) {
	p.results = append(p.results, r)
}

func readFromTestFile(name string, pub data.Publisher) {
	f := openFile(name)
	s, err := filelogsource.New(f, time.Time{}, 2020)
	errorutil.MustSucceed(err)
	r := logsource.NewReader(s, pub)
	r.Run()
}

func TestTrackingFromFiles(t *testing.T) {
	Convey("Tracking From Files", t, func() {
		resultPublisher := &fakeResultPublisher{}

		newTempTracker := func() (*Tracker, func()) {
			dir, clearDir := testutil.TempDir(t)
			t, err := New(dir, resultPublisher)
			So(err, ShouldBeNil)

			return t, func() {
				So(t.Close(), ShouldBeNil)
				clearDir()
			}
		}

		t, clear := newTempTracker()
		defer clear()

		done, cancel := t.Run()

		// TODO: check each field of the results for the right values!!!

		Convey("Single bounced message", func() {
			readFromTestFile("test_files/1_bounce_simple.log", t.Publisher())
			cancel()
			done()

			So(len(resultPublisher.results), ShouldEqual, 1)
			So(resultPublisher.results[0][QueueSenderLocalPartKey], ShouldEqual, "user")
			So(resultPublisher.results[0][QueueSenderDomainPartKey], ShouldEqual, "sender.com")
			So(resultPublisher.results[0][QueueMessageIDKey], ShouldEqual, "ca10035e-2951-bfd5-ec7e-1a5773fce1cd@mail.sender.com")
			So(resultPublisher.results[0][ResultRecipientLocalPartKey], ShouldEqual, "invalid.email")
			So(resultPublisher.results[0][ResultRecipientDomainPartKey], ShouldEqual, "example.com")
			So(resultPublisher.results[0][ResultMessageDirectionKey], ShouldEqual, MessageDirectionOutbound)
		})

		Convey("Five messages, two bounced", func() {
			readFromTestFile("test_files/2_multiple_recipieints_some_bounces.log", t.Publisher())
			cancel()
			done()
			So(len(resultPublisher.results), ShouldEqual, 5)
		})

		Convey("One message deliered locally", func() {
			readFromTestFile("test_files/3_local_delivery.log", t.Publisher())
			cancel()
			done()
			So(len(resultPublisher.results), ShouldEqual, 1)
			So(resultPublisher.results[0][ResultMessageDirectionKey], ShouldEqual, MessageDirectionIncoming)
		})

		Convey("Local queue ADCC76373 is forever lost by postfix (not delivered)", func() {
			// Sometimes postfix moves messages to a local (outbound) queue that
			// is never processed again (being lost), so we basically ignore it.
			// In the future, the "Message Detective" should be able to track such lost
			// messages.
			readFromTestFile("test_files/4_lost_queue.log", t.Publisher())
			cancel()
			done()
			So(len(resultPublisher.results), ShouldEqual, 3)
		})

		Convey("A mail sent with zimbra and amavisd", func() {
			// A more complex postfix setup, using amavisd and zimbra.
			// There are extra steps and the message is moved around on different queues.
			// Sometimes postfix moves messages to a local (outbound) queue that
			// There's also usage of NOQUEUE, which is a non existent queue, never removed.
			readFromTestFile("test_files/5_zimbra_amavisd.log", t.Publisher())
			cancel()
			done()
			So(len(resultPublisher.results), ShouldEqual, 1)
			So(resultPublisher.results[0][ResultRecipientLocalPartKey], ShouldEqual, "recipient")
			So(resultPublisher.results[0][ResultRecipientDomainPartKey], ShouldEqual, "recipient.example.com")
			So(resultPublisher.results[0][QueueSenderLocalPartKey], ShouldEqual, "sender")
			So(resultPublisher.results[0][QueueSenderDomainPartKey], ShouldEqual, "sender.example.com")

			// TODO: We are at the moment unable to track how the connection started as we are not able
			// to process NOQUEUE!!!
			//So(resultPublisher.results[0][ConnectionBeginKey], ShouldNotBeNil)
		})

	})
}
