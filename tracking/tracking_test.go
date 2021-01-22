package tracking

import (
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/data"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/logeater/filelogsource"
	"gitlab.com/lightmeter/controlcenter/logeater/logsource"
	parser "gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"io"
	"log"
	"os"
	"strings"
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

func readFromTestReader(reader io.Reader, pub data.Publisher) {
	s, err := filelogsource.New(reader, time.Time{}, 2020)
	errorutil.MustSucceed(err)
	r := logsource.NewReader(s, pub)
	r.Run()
}

func readFromTestFile(name string, pub data.Publisher) {
	f := openFile(name)
	readFromTestReader(f, pub)
}

func readFromTestContent(content string, pub data.Publisher) {
	r := strings.NewReader(content)
	readFromTestReader(r, pub)
}

func buildPublisherAndTempTracker(t *testing.T) (*fakeResultPublisher, *Tracker, func()) {
	pub := &fakeResultPublisher{}

	dir, clearDir := testutil.TempDir(t)
	tracker, err := New(dir, pub)
	So(err, ShouldBeNil)

	log.Println("Log Dir:", dir)

	return pub, tracker, func() {
		So(tracker.Close(), ShouldBeNil)
		clearDir()
	}
}

func TestMostRecentLogTime(t *testing.T) {
	Convey("Obtain most recent time", t, func() {
		_, t, clear := buildPublisherAndTempTracker(t)
		defer clear()
		done, cancel := t.Run()

		Convey("Nothing read", func() {
			cancel()
			done()
			So(t.MostRecentLogTime(), ShouldResemble, time.Time{})
		})

		Convey("File with a connection", func() {
			readFromTestContent(`Oct 13 16:40:39 ucs postfix/smtpd[18568]: connect from unknown[28.55.140.112]`, t.Publisher())
			cancel()
			done()
			So(t.MostRecentLogTime(), ShouldResemble, testutil.MustParseTime(`2020-10-13 16:40:39 +0000`))
		})
	})
}

func TestTrackingFromUnsupportedLogFiles(t *testing.T) {
	// TODO: support those files!
	Convey("Some strange and for now unsupported log lines, that need to be supported in the future!", t, func() {
		pub, t, clear := buildPublisherAndTempTracker(t)
		defer clear()
		done, cancel := t.Run()

		Convey("Unsupported lines, with weird clone syntax", func() {
			readFromTestFile("test_files/8_weird_log_file.log", t.Publisher())
			cancel()
			done()
			So(len(pub.results), ShouldEqual, 0)
		})
	})
}

func TestTrackingFromFiles(t *testing.T) {
	Convey("Tracking From Files", t, func() {
		pub, t, clear := buildPublisherAndTempTracker(t)
		defer clear()

		done, cancel := t.Run()

		countResults := func() int {
			var count int
			err := t.dbconn.RoConn.QueryRow(`select count(*) from results`).Scan(&count)
			So(err, ShouldBeNil)
			return count
		}

		countResultData := func() int {
			var count int
			err := t.dbconn.RoConn.QueryRow(`select count(*) from result_data`).Scan(&count)
			So(err, ShouldBeNil)
			return count
		}

		countQueues := func() int {
			var count int
			err := t.dbconn.RoConn.QueryRow(`select count(*) from queues`).Scan(&count)
			So(err, ShouldBeNil)
			return count
		}

		countQueueData := func() int {
			var count int
			err := t.dbconn.RoConn.QueryRow(`select count(*) from queue_data`).Scan(&count)
			So(err, ShouldBeNil)
			return count
		}

		countConnections := func() int {
			var count int
			err := t.dbconn.RoConn.QueryRow(`select count(*) from connections`).Scan(&count)
			So(err, ShouldBeNil)
			return count
		}

		countConnectionData := func() int {
			var count int
			err := t.dbconn.RoConn.QueryRow(`select count(*) from connection_data`).Scan(&count)
			So(err, ShouldBeNil)
			return count
		}

		countPids := func() int {
			var count int
			err := t.dbconn.RoConn.QueryRow(`select count(*) from pids`).Scan(&count)
			So(err, ShouldBeNil)
			return count
		}

		countMessageIds := func() int {
			var count int
			err := t.dbconn.RoConn.QueryRow(`select count(*) from messageids`).Scan(&count)
			So(err, ShouldBeNil)
			return count
		}

		// TODO: check each field of the results for the right values!!!

		Convey("With Tracker", func() {
			Convey("Single bounced message", func() {
				readFromTestFile("test_files/1_bounce_simple.log", t.Publisher())
				cancel()
				done()

				// the second message is a bounce back one
				So(len(pub.results), ShouldEqual, 2)
				So(pub.results[0][QueueSenderLocalPartKey], ShouldEqual, "user")
				So(pub.results[0][QueueSenderDomainPartKey], ShouldEqual, "sender.com")
				So(pub.results[0][QueueMessageIDKey], ShouldEqual, "ca10035e-2951-bfd5-ec7e-1a5773fce1cd@mail.sender.com")
				So(pub.results[0][ResultRecipientLocalPartKey], ShouldEqual, "invalid.email")
				So(pub.results[0][ResultRecipientDomainPartKey], ShouldEqual, "example.com")
				So(pub.results[0][ResultMessageDirectionKey], ShouldEqual, MessageDirectionOutbound)
				So(pub.results[0][ResultStatusKey], ShouldEqual, parser.BouncedStatus)

				So(countQueues(), ShouldEqual, 0)
				So(countQueueData(), ShouldEqual, 0)
				So(countConnections(), ShouldEqual, 0)
				So(countConnectionData(), ShouldEqual, 0)
				So(countPids(), ShouldEqual, 0)
				So(countMessageIds(), ShouldEqual, 0)
			})

			Convey("Five messages, two bounced", func() {
				Convey("Complete log, with last 'remove' available", func() {
					readFromTestFile("test_files/2_multiple_recipieints_some_bounces.log", t.Publisher())
					cancel()
					done()
					So(len(pub.results), ShouldEqual, 6)

					So(countQueues(), ShouldEqual, 0)
					So(countQueueData(), ShouldEqual, 0)
					So(countConnections(), ShouldEqual, 0)
					So(countConnectionData(), ShouldEqual, 0)
					So(countPids(), ShouldEqual, 0)
					So(countMessageIds(), ShouldEqual, 0)
				})

				Convey("Complete log, with last 'remove' missing", func() {
					readFromTestFile("test_files/2_multiple_recipieints_some_bounces_no_last_remove.log", t.Publisher())
					cancel()
					done()
					So(len(pub.results), ShouldEqual, 6)
				})
			})

			Convey("One message deliered locally", func() {
				readFromTestFile("test_files/3_local_delivery.log", t.Publisher())
				cancel()
				done()
				So(len(pub.results), ShouldEqual, 1)
				So(pub.results[0][ResultMessageDirectionKey], ShouldEqual, MessageDirectionIncoming)
			})

			Convey("Local queue ADCC76373 is forever lost by postfix (not delivered)", func() {
				// Sometimes postfix moves messages to a local (outbound) queue that
				// is never processed again (being lost), so we basically ignore it.
				// In the future, the "Message Detective" should be able to track such lost
				// messages.
				readFromTestFile("test_files/4_lost_queue.log", t.Publisher())
				cancel()
				done()
				So(len(pub.results), ShouldEqual, 3)
			})

			Convey("A mail sent with zimbra and amavisd", func() {
				// A more complex postfix setup, using amavisd and zimbra.
				// There are extra steps and the message is moved around on different queues.
				// Sometimes postfix moves messages to a local (outbound) queue that
				// There's also usage of NOQUEUE, which is a non existent queue, never removed.
				readFromTestFile("test_files/5_zimbra_amavisd.log", t.Publisher())
				cancel()
				done()
				So(len(pub.results), ShouldEqual, 1)
				So(pub.results[0][ResultRecipientLocalPartKey], ShouldEqual, "recipient")
				So(pub.results[0][ResultRecipientDomainPartKey], ShouldEqual, "recipient.example.com")
				So(pub.results[0][QueueSenderLocalPartKey], ShouldEqual, "sender")
				So(pub.results[0][QueueSenderDomainPartKey], ShouldEqual, "sender.example.com")

				// TODO: We are at the moment unable to track how the connection started as we are not able
				// to process NOQUEUE!!!
				//So(pub.results[0][ConnectionBeginKey], ShouldNotBeNil)
			})

			Convey("An e-mail gets deferred", func() {
				readFromTestFile("test_files/6_deferred_message_retry.log", t.Publisher())
				cancel()
				done()
				So(len(pub.results), ShouldEqual, 2)

				So(pub.results[0][ResultRecipientLocalPartKey], ShouldEqual, "recipient")
				So(pub.results[0][ResultRecipientDomainPartKey], ShouldEqual, "recipient.com")
				So(pub.results[0][ResultStatusKey], ShouldEqual, parser.DeferredStatus)
				So(pub.results[0][ResultMessageDirectionKey], ShouldEqual, MessageDirectionOutbound)

				So(pub.results[1][ResultStatusKey], ShouldEqual, parser.SentStatus)
				So(pub.results[1][ResultMessageDirectionKey], ShouldEqual, MessageDirectionOutbound)
			})

			Convey("Log with only connections and disconnections. No queues are created", func() {
				readFromTestFile("test_files/7_only_connections_and_disconnections.log", t.Publisher())
				cancel()
				done()
				So(len(pub.results), ShouldEqual, 0)

				So(countQueues(), ShouldEqual, 0)
				So(countQueueData(), ShouldEqual, 0)
				So(countConnections(), ShouldEqual, 0)
				So(countConnectionData(), ShouldEqual, 0)
				So(countPids(), ShouldEqual, 0)
				So(countMessageIds(), ShouldEqual, 0)
			})
		})

		// we expected all results to have been consumed
		So(countResults(), ShouldEqual, 0)
		So(countResultData(), ShouldEqual, 0)
	})
}
