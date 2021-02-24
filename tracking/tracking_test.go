// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package tracking

import (
	"bytes"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/pkg/postfix"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/logeater/filelogsource"
	"gitlab.com/lightmeter/controlcenter/logeater/logsource"
	parser "gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"
)

var _ = parser.Parse

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

func readFromTestReader(reader io.Reader, pub postfix.Publisher) {
	s, err := filelogsource.New(reader, time.Time{}, 2020)
	errorutil.MustSucceed(err)
	r := logsource.NewReader(s, pub)
	r.Run()
}

func readFromTestFile(name string, pub postfix.Publisher) {
	f := openFile(name)
	readFromTestReader(f, pub)
}

func readFromTestContent(content string, pub postfix.Publisher) {
	r := strings.NewReader(content)
	readFromTestReader(r, pub)
}

func buildPublisherAndTempTracker(t *testing.T) (*fakeResultPublisher, *Tracker, func()) {
	pub := &fakeResultPublisher{}

	dir, clearDir := testutil.TempDir(t)
	tracker, err := New(dir, pub)
	So(err, ShouldBeNil)

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
			// Somehow it generates one message, but this is really not that supported at the moment!
			So(len(pub.results), ShouldEqual, 1)
		})
	})
}

func computeLineOffsets(b []byte) []int {
	offsets := []int{}

	i := 0

	for {
		index := bytes.IndexByte(b[i:], byte('\n'))

		if index == -1 {
			break
		}

		i += index + 1

		offsets = append(offsets, i)
	}

	return offsets
}

var _ = ioutil.TempDir

func TestReadingFromArbitraryLines(t *testing.T) {
	Convey("Reading from arbitrary lines", t, func() {
		file, err := os.Open("test_files/1_bounce_simple.log")
		So(err, ShouldBeNil)

		b, err := ioutil.ReadAll(file)
		So(err, ShouldBeNil)

		offsets := computeLineOffsets(b)

		// Start reading from arbitrary lines in the file
		for _, offset := range offsets {
			_, t, clear := buildPublisherAndTempTracker(t)
			defer clear()

			done, cancel := t.Run()

			content := b[offset:]
			r := bytes.NewReader(content)

			readFromTestReader(r, t.Publisher())
			cancel()
			done()
		}
	})
}

func TestTrackingFromFiles(t *testing.T) {
	Convey("Tracking From Files", t, func() {
		pub, t, clear := buildPublisherAndTempTracker(t)
		_ = pub
		defer clear()

		done, cancel := t.Run()

		queryConn, release := t.dbconn.RoConnPool.Acquire()

		defer release()

		countResults := func() int {
			var count int
			err := queryConn.QueryRow(`select count(*) from results`).Scan(&count)
			So(err, ShouldBeNil)
			return count
		}

		_ = countResults

		countResultData := func() int {
			var count int
			err := queryConn.QueryRow(`select count(*) from result_data`).Scan(&count)
			So(err, ShouldBeNil)
			return count
		}

		_ = countResultData

		countQueues := func() int {
			var count int
			err := queryConn.QueryRow(`select count(*) from queues`).Scan(&count)
			So(err, ShouldBeNil)
			return count
		}

		_ = countQueues

		countQueueData := func() int {
			var count int
			err := queryConn.QueryRow(`select count(*) from queue_data`).Scan(&count)
			So(err, ShouldBeNil)
			return count
		}

		_ = countQueueData

		countConnections := func() int {
			var count int
			err := queryConn.QueryRow(`select count(*) from connections`).Scan(&count)
			So(err, ShouldBeNil)
			return count
		}

		_ = countConnections

		countConnectionData := func() int {
			var count int
			err := queryConn.QueryRow(`select count(*) from connection_data`).Scan(&count)
			So(err, ShouldBeNil)
			return count
		}

		_ = countConnectionData

		countPids := func() int {
			var count int
			err := queryConn.QueryRow(`select count(*) from pids`).Scan(&count)
			So(err, ShouldBeNil)
			return count
		}

		_ = countPids

		// TODO: check each field of the results for the right values!!!

		Convey("With Tracker", func() {
			Convey("Well behaving files", func() {
				Convey("Single bounced message", func() {
					readFromTestFile("test_files/1_bounce_simple.log", t.Publisher())
					cancel()
					done()

					// the second message is a bounce back one
					So(len(pub.results), ShouldEqual, 2)
					So(pub.results[0][ConnectionClientHostnameKey].Text(), ShouldEqual, "some.domain.name")
					So(pub.results[0][QueueSenderLocalPartKey].Text(), ShouldEqual, "user")
					So(pub.results[0][QueueSenderDomainPartKey].Text(), ShouldEqual, "sender.com")
					So(pub.results[0][QueueMessageIDKey].Text(), ShouldEqual, "ca10035e-2951-bfd5-ec7e-1a5773fce1cd@mail.sender.com")
					So(pub.results[0][QueueOriginalMessageSizeKey].Int64(), ShouldEqual, 391)
					So(pub.results[0][QueueProcessedMessageSizeKey].Int64(), ShouldEqual, 1111)
					So(pub.results[0][ResultRecipientLocalPartKey].Text(), ShouldEqual, "invalid.email")
					So(pub.results[0][ResultRecipientDomainPartKey].Text(), ShouldEqual, "example.com")
					So(pub.results[0][ResultMessageDirectionKey].Int64(), ShouldEqual, MessageDirectionOutbound)
					So(pub.results[0][ResultStatusKey].Int64(), ShouldEqual, parser.BouncedStatus)

					So(countQueues(), ShouldEqual, 0)
					So(countQueueData(), ShouldEqual, 0)
					So(countConnections(), ShouldEqual, 0)
					So(countConnectionData(), ShouldEqual, 0)
					So(countPids(), ShouldEqual, 0)
				})

				Convey("Five messages, two bounced", func() {
					Convey("Complete log, with last 'remove' available", func() {
						// FIXME: this test sporadically misbehaves and fails when the number of notifiers is > 1
						// - some results (if not all!) fail to be notified, as the deletion for their queues fail!
						readFromTestFile("test_files/2_multiple_recipients_some_bounces.log", t.Publisher())
						cancel()
						done()
						So(len(pub.results), ShouldEqual, 6)

						So(countQueues(), ShouldEqual, 0)
						So(countQueueData(), ShouldEqual, 0)
						So(countConnections(), ShouldEqual, 0)
						So(countConnectionData(), ShouldEqual, 0)
						So(countPids(), ShouldEqual, 0)
					})

					Convey("Complete log, with last 'remove' missing", func() {
						readFromTestFile("test_files/2_multiple_recipients_some_bounces_no_last_remove.log", t.Publisher())
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
					So(pub.results[0][ResultMessageDirectionKey].Int64(), ShouldEqual, MessageDirectionIncoming)
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
					So(pub.results[0][ResultRecipientLocalPartKey].Text(), ShouldEqual, "recipient")
					So(pub.results[0][ResultRecipientDomainPartKey].Text(), ShouldEqual, "recipient.example.com")
					So(pub.results[0][QueueSenderLocalPartKey].Text(), ShouldEqual, "sender")
					So(pub.results[0][QueueSenderDomainPartKey].Text(), ShouldEqual, "sender.example.com")

					// TODO: We are at the moment unable to track how the connection started as we are not able
					// to process NOQUEUE!!!
					//So(pub.results[0][ConnectionBeginKey], ShouldNotBeNil)
				})

				Convey("An e-mail gets deferred", func() {
					readFromTestFile("test_files/6_deferred_message_retry.log", t.Publisher())
					cancel()
					done()
					So(len(pub.results), ShouldEqual, 2)

					So(pub.results[0][ResultRecipientLocalPartKey].Text(), ShouldEqual, "recipient")
					So(pub.results[0][ResultRecipientDomainPartKey].Text(), ShouldEqual, "recipient.com")
					So(pub.results[0][ResultStatusKey].Int64(), ShouldEqual, parser.DeferredStatus)
					So(pub.results[0][ResultMessageDirectionKey].Int64(), ShouldEqual, MessageDirectionOutbound)

					So(pub.results[1][ResultStatusKey].Int64(), ShouldEqual, parser.SentStatus)
					So(pub.results[1][ResultMessageDirectionKey].Int64(), ShouldEqual, MessageDirectionOutbound)
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
				})

				Convey("Two messages are sent. The first one for one destination and bounces, and the second one to multiples destinations", func() {
					readFromTestFile("test_files/9_mixed_messages.log", t.Publisher())
					cancel()
					done()

					So(len(pub.results), ShouldEqual, 8)

					So(countQueues(), ShouldEqual, 0)
					So(countQueueData(), ShouldEqual, 0)
					So(countConnections(), ShouldEqual, 0)
					So(countConnectionData(), ShouldEqual, 0)
					So(countPids(), ShouldEqual, 0)
				})

				Convey("Pickup in action", func() {
					readFromTestFile("test_files/10_pickup.log", t.Publisher())
					cancel()
					done()

					So(len(pub.results), ShouldEqual, 1)

					So(countQueues(), ShouldEqual, 0)
					So(countQueueData(), ShouldEqual, 0)
					So(countConnections(), ShouldEqual, 0)
					So(countConnectionData(), ShouldEqual, 0)
					So(countPids(), ShouldEqual, 0)
				})

				Convey("Single delived message", func() {
					readFromTestFile("test_files/11_single_successful_delivery.log", t.Publisher())
					cancel()
					done()

					So(len(pub.results), ShouldEqual, 1)

					So(pub.results[0][ConnectionClientHostnameKey].Text(), ShouldEqual, "client.example.com")
					So(pub.results[0][QueueSenderLocalPartKey].Text(), ShouldEqual, "sender")
					So(pub.results[0][QueueSenderDomainPartKey].Text(), ShouldEqual, "mydomain.com")
					So(pub.results[0][QueueMessageIDKey].Text(), ShouldEqual, "264dc34c-ad52-466c-6d41-6622dfced3b8@mydomain.com")
					So(pub.results[0][QueueOriginalMessageSizeKey].Int64(), ShouldEqual, 502)
					So(pub.results[0][QueueProcessedMessageSizeKey].Int64(), ShouldEqual, 1188)
					So(pub.results[0][ResultRecipientLocalPartKey].Text(), ShouldEqual, "recipient1")
					So(pub.results[0][ResultRecipientDomainPartKey].Text(), ShouldEqual, "dst1.example.com")
					So(pub.results[0][ResultMessageDirectionKey].Int64(), ShouldEqual, MessageDirectionOutbound)
					So(pub.results[0][ResultStatusKey].Int64(), ShouldEqual, parser.SentStatus)

					So(countQueues(), ShouldEqual, 0)
					So(countQueueData(), ShouldEqual, 0)
					So(countConnections(), ShouldEqual, 0)
					So(countConnectionData(), ShouldEqual, 0)
					So(countPids(), ShouldEqual, 0)
				})

				Convey("Two deliveries using the same smtp2 pid, processing in order", func() {
					readFromTestFile("test_files/12_two_independent_deliveries_in_the_same_smtpd_process_in_order.log", t.Publisher())
					cancel()
					done()

					So(len(pub.results), ShouldEqual, 2)

					So(pub.results[0][ConnectionClientHostnameKey].Text(), ShouldEqual, "client.example.com")
					So(pub.results[0][QueueSenderLocalPartKey].Text(), ShouldEqual, "sender")
					So(pub.results[0][QueueSenderDomainPartKey].Text(), ShouldEqual, "mydomain.com")
					So(pub.results[0][QueueOriginalMessageSizeKey].Int64(), ShouldEqual, 502)
					So(pub.results[0][QueueProcessedMessageSizeKey].Int64(), ShouldEqual, 1188)
					So(pub.results[0][ResultRecipientLocalPartKey].Text(), ShouldEqual, "recipient1")
					So(pub.results[0][ResultRecipientDomainPartKey].Text(), ShouldEqual, "dst1.example.com")
					So(pub.results[0][ResultMessageDirectionKey].Int64(), ShouldEqual, MessageDirectionOutbound)
					So(pub.results[0][ResultStatusKey].Int64(), ShouldEqual, parser.SentStatus)
					So(pub.results[0][QueueMessageIDKey].Text(), ShouldEqual, "264dc34c-ad52-466c-6d41-6622dfced3b8@mydomain.com")

					So(pub.results[1][ConnectionClientHostnameKey].Text(), ShouldEqual, "client2.another.example.com")
					So(pub.results[1][QueueSenderLocalPartKey].Text(), ShouldEqual, "sender2")
					So(pub.results[1][QueueSenderDomainPartKey].Text(), ShouldEqual, "mydomain2.com")
					So(pub.results[1][QueueOriginalMessageSizeKey].Int64(), ShouldEqual, 503)
					So(pub.results[1][QueueProcessedMessageSizeKey].Int64(), ShouldEqual, 1189)
					So(pub.results[1][ResultRecipientLocalPartKey].Text(), ShouldEqual, "dst2")
					So(pub.results[1][ResultRecipientDomainPartKey].Text(), ShouldEqual, "dst2.com")
					So(pub.results[1][ResultMessageDirectionKey].Int64(), ShouldEqual, MessageDirectionOutbound)
					So(pub.results[1][ResultStatusKey].Int64(), ShouldEqual, parser.SentStatus)
					So(pub.results[1][QueueMessageIDKey].Text(), ShouldEqual, "lalalacacaca@lala.com")

					So(countQueues(), ShouldEqual, 0)
					So(countQueueData(), ShouldEqual, 0)
					So(countConnections(), ShouldEqual, 0)
					So(countConnectionData(), ShouldEqual, 0)
					So(countPids(), ShouldEqual, 0)
				})

				Convey("Two deliveries using the same smtp2 pid, processing mixed", func() {
					readFromTestFile("test_files/12_two_independent_deliveries_in_the_same_smtpd_process_mixed.log", t.Publisher())
					cancel()
					done()

					So(len(pub.results), ShouldEqual, 2)

					// the later message is sent before the second one

					So(pub.results[0][ConnectionClientHostnameKey].Text(), ShouldEqual, "client2.another.example.com")
					So(pub.results[0][QueueSenderLocalPartKey].Text(), ShouldEqual, "sender2")
					So(pub.results[0][QueueSenderDomainPartKey].Text(), ShouldEqual, "mydomain2.com")
					So(pub.results[0][QueueOriginalMessageSizeKey].Int64(), ShouldEqual, 503)
					So(pub.results[0][QueueProcessedMessageSizeKey].Int64(), ShouldEqual, 1189)
					So(pub.results[0][ResultRecipientLocalPartKey].Text(), ShouldEqual, "dst2")
					So(pub.results[0][ResultRecipientDomainPartKey].Text(), ShouldEqual, "dst2.com")
					So(pub.results[0][ResultMessageDirectionKey].Int64(), ShouldEqual, MessageDirectionOutbound)
					So(pub.results[0][ResultStatusKey].Int64(), ShouldEqual, parser.SentStatus)
					So(pub.results[0][QueueMessageIDKey].Text(), ShouldEqual, "lalalacacaca@lala.com")

					So(pub.results[1][ConnectionClientHostnameKey].Text(), ShouldEqual, "client.example.com")
					So(pub.results[1][QueueSenderLocalPartKey].Text(), ShouldEqual, "sender")
					So(pub.results[1][QueueSenderDomainPartKey].Text(), ShouldEqual, "mydomain.com")
					So(pub.results[1][QueueOriginalMessageSizeKey].Int64(), ShouldEqual, 502)
					So(pub.results[1][QueueProcessedMessageSizeKey].Int64(), ShouldEqual, 1188)
					So(pub.results[1][ResultRecipientLocalPartKey].Text(), ShouldEqual, "recipient1")
					So(pub.results[1][ResultRecipientDomainPartKey].Text(), ShouldEqual, "dst1.example.com")
					So(pub.results[1][ResultMessageDirectionKey].Int64(), ShouldEqual, MessageDirectionOutbound)
					So(pub.results[1][ResultStatusKey].Int64(), ShouldEqual, parser.SentStatus)
					So(pub.results[1][QueueMessageIDKey].Text(), ShouldEqual, "264dc34c-ad52-466c-6d41-6622dfced3b8@mydomain.com")

					So(countQueues(), ShouldEqual, 0)
					So(countQueueData(), ShouldEqual, 0)
					So(countConnections(), ShouldEqual, 0)
					So(countConnectionData(), ShouldEqual, 0)
					So(countPids(), ShouldEqual, 0)
				})

				Convey("Initial queue msgid can be empty (issue #388)", func() {
					readFromTestFile("test_files/13_empty_msgid_issue_388.log", t.Publisher())
					cancel()
					done()

					So(len(pub.results), ShouldEqual, 1)

					// the later message is sent before the second one

					So(pub.results[0][ConnectionClientHostnameKey].Text(), ShouldEqual, "h-2ee7ba9722900c79")
					So(pub.results[0][QueueSenderLocalPartKey].Text(), ShouldEqual, "h-19132c")
					So(pub.results[0][QueueSenderDomainPartKey].Text(), ShouldEqual, "h-e858bb21f.com")
					So(pub.results[0][QueueOriginalMessageSizeKey].Int64(), ShouldEqual, 1209)
					So(pub.results[0][QueueProcessedMessageSizeKey].Int64(), ShouldEqual, 2429)
					So(pub.results[0][ResultRecipientLocalPartKey].Text(), ShouldEqual, "h-10")
					So(pub.results[0][ResultRecipientDomainPartKey].Text(), ShouldEqual, "h-e858bb21f.com")
					So(pub.results[0][ResultMessageDirectionKey].Int64(), ShouldEqual, MessageDirectionIncoming)
					So(pub.results[0][ResultStatusKey].Int64(), ShouldEqual, parser.SentStatus)
					So(pub.results[0][QueueDeliveryNameKey].Text(), ShouldEqual, "CACACACA")
					So(pub.results[0][QueueMessageIDKey].Text(), ShouldEqual, "h-58c98222ea74bdf467d69d856d@h-028957b9aefc40.com")

					So(countQueues(), ShouldEqual, 0)
					So(countQueueData(), ShouldEqual, 0)
					So(countConnections(), ShouldEqual, 0)
					So(countConnectionData(), ShouldEqual, 0)
					So(countPids(), ShouldEqual, 0)
				})
			})

			// we expected all results to have been consumed
			So(countResults(), ShouldEqual, 0)
			So(countResultData(), ShouldEqual, 0)
		})

		Convey("Files with unsupported behaviour (to be investigated)", func() {
			Convey("Queue is reused (BABABABABA)", func() {
				// FIXME: right now we are ignoring the error happening in this file probably
				// due the reuse of an already closed queue,
				// but this use case should be supported as it seems to happen quite often
				readFromTestFile("test_files/14_reuse_of_queueid.log", t.Publisher())
				cancel()
				done()
			})
		})
	})
}
