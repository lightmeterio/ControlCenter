package logdb

import (
	"context"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/dashboard"
	"gitlab.com/lightmeter/controlcenter/data"
	"gitlab.com/lightmeter/controlcenter/domainmapping"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	parser "gitlab.com/lightmeter/postfix-log-parser"
)

func init() {
	// NOTE: unfortunately the domain mapping code uses a singleton (to be accessed internally via sqlite)
	// that outlives all the test cases, so it's more clear for it to be defined globally as well
	m, err := domainmapping.Mapping(domainmapping.RawList{"grouped": []string{"domaintobegrouped.com", "domaintobegrouped.de"}})
	errorutil.MustSucceed(err)

	lmsqlite3.Initialize(lmsqlite3.Options{
		"domain_mapping": &m,
	})
}

func TestDatabaseCreation(t *testing.T) {
	Convey("Creation fails on several scenarios", t, func() {
		Convey("No Permission on workspace", func() {
			// FIXME: this is relying on linux properties, as /proc is a read-only directory
			_, err := Open("/proc/lalala", Config{Location: time.UTC})
			So(err, ShouldNotBeNil)
		})

		Convey("Db is a directory instead of a file", func() {
			dir, clearDir := testutil.TempDir()
			defer clearDir()
			So(os.Mkdir(path.Join(dir, "logs.db"), os.ModePerm), ShouldBeNil)
			_, err := Open(dir, Config{Location: time.UTC})
			So(err, ShouldNotBeNil)
		})

		Convey("Db is not a sqlite file", func() {
			dir, clearDir := testutil.TempDir()
			defer clearDir()
			ioutil.WriteFile(path.Join(dir, "logs.db"), []byte("not a sqlite file header"), os.ModePerm)
			_, err := Open(dir, Config{Location: time.UTC})
			So(err, ShouldNotBeNil)
		})
	})

	Convey("Creation succeeds", t, func() {
		Convey("Create DB", func() {
			dir, clearDir := testutil.TempDir()
			defer clearDir()
			db, err := Open(dir, Config{Location: time.UTC})
			So(err, ShouldBeNil)

			defer db.Close()
			So(db.HasLogs(), ShouldBeFalse)
		})

		Convey("Empty Database is properly closed", func() {
			dir, clearDir := testutil.TempDir()
			defer clearDir()
			db, err := Open(dir, Config{Location: time.UTC})
			So(err, ShouldBeNil)
			So(db.HasLogs(), ShouldBeFalse)
			So(db.Close(), ShouldBeNil)
		})

		Convey("Reopening workspace succeeds", func() {
			dir, clearDir := testutil.TempDir()
			defer clearDir()

			ws1, err := Open(dir, Config{Location: time.UTC})
			ws1.Close()

			ws2, err := Open(dir, Config{Location: time.UTC})
			So(err, ShouldBeNil)
			ws2.Close()
		})
	})
}

func parseTimeInterval(from, to string) data.TimeInterval {
	interval, err := data.ParseTimeInterval(from, to, time.UTC)
	if err != nil {
		panic("pasring interval")
	}
	return interval
}

var (
	dummyContext = context.Background()
)

func countByStatus(dashboard dashboard.Dashboard, status parser.SmtpStatus, interval data.TimeInterval) int {
	v, err := dashboard.CountByStatus(dummyContext, status, interval)
	So(err, ShouldBeNil)
	return v
}

func topBusiestDomains(dashboard dashboard.Dashboard, interval data.TimeInterval) dashboard.Pairs {
	pairs, err := dashboard.TopBusiestDomains(dummyContext, interval)
	So(err, ShouldBeNil)
	return pairs
}

func topBouncedDomains(dashboard dashboard.Dashboard, interval data.TimeInterval) dashboard.Pairs {
	pairs, err := dashboard.TopBouncedDomains(dummyContext, interval)
	So(err, ShouldBeNil)
	return pairs
}

func topDeferredDomains(dashboard dashboard.Dashboard, interval data.TimeInterval) dashboard.Pairs {
	pairs, err := dashboard.TopDeferredDomains(dummyContext, interval)
	So(err, ShouldBeNil)
	return pairs
}

func deliveryStatus(dashboard dashboard.Dashboard, interval data.TimeInterval) dashboard.Pairs {
	pairs, err := dashboard.DeliveryStatus(dummyContext, interval)
	So(err, ShouldBeNil)
	return pairs
}

func TestLogsInsertion(t *testing.T) {
	Convey("LogInsertion", t, func() {
		dir, clearDir := testutil.TempDir()
		defer clearDir()

		buildWs := func() (DB, <-chan interface{}, data.Publisher, dashboard.Dashboard, func()) {
			db, err := Open(dir, Config{Location: time.UTC})
			So(err, ShouldBeNil)
			done := db.Run()
			pub := db.NewPublisher()
			dashboard, err := dashboard.New(db.ReadConnection())
			So(err, ShouldBeNil)

			return db, done, pub, dashboard, func() {
				So(dashboard.Close(), ShouldBeNil)
				So(db.Close(), ShouldBeNil)
			}
		}

		var converter parser.TimeConverter

		initConverter := func(year int, db *DB) {
			t := func(ts time.Time) time.Time {
				if !ts.IsZero() {
					return ts
				}

				return parser.DefaultTimeInYear(year, ts.Location())
			}(db.MostRecentLogTime())

			converter = parser.NewTimeConverter(t, func(int, parser.Time, parser.Time) {})
		}

		smtpStatusRecordWithRecipient := func(status parser.SmtpStatus, t parser.Time, recipientLocalPart, recipientDomainPart string) data.Record {
			So(converter, ShouldNotBeNil)

			return data.Record{
				Time: converter.Convert(t),
				Header: parser.Header{
					Time:    t,
					Host:    "mail",
					Process: "smtp",
				},
				Payload: parser.SmtpSentStatus{
					Queue:               "AA",
					RecipientLocalPart:  recipientLocalPart,
					RecipientDomainPart: recipientDomainPart,
					RelayName:           "",
					RelayIP:             nil,
					RelayPort:           0,
					Delay:               3.4,
					Delays:              parser.Delays{Smtpd: 0.1, Cleanup: 0.2, Qmgr: 0.3},
					Dsn:                 "4.5.6",
					Status:              status,
					ExtraMessage:        "",
				},
			}
		}

		smtpStatusRecord := func(status parser.SmtpStatus, t parser.Time) data.Record {
			return smtpStatusRecordWithRecipient(status, t, "recipient", "test.com")
		}

		noPayloadRecord := func(t parser.Time) data.Record {
			So(converter, ShouldNotBeNil)

			return data.Record{
				Time: converter.Convert(t),
				Header: parser.Header{
					Time:    t,
					Host:    "mail",
					Process: "smtp",
				},
				Payload: nil,
			}
		}

		t := func(mo time.Month, d, h, m, s uint8) parser.Time {
			return parser.Time{Month: mo, Day: d, Hour: h, Minute: m, Second: s}
		}

		Convey("Inserting logs", func() {
			Convey("Inserts nothing", func() {
				db, done, pub, dashboard, dtor := buildWs()
				defer dtor()
				pub.Close()
				<-done

				initConverter(1999, &db)

				interval := parseTimeInterval("1999-12-02", "2000-01-03")

				So(db.HasLogs(), ShouldBeFalse)
				So(countByStatus(dashboard, parser.BouncedStatus, interval), ShouldEqual, 0)
				So(countByStatus(dashboard, parser.DeferredStatus, interval), ShouldEqual, 0)
				So(countByStatus(dashboard, parser.SentStatus, interval), ShouldEqual, 0)
			})

			Convey("Inserts one log entry", func() {
				db, done, pub, dashboard, dtor := buildWs()
				defer dtor()

				initConverter(1999, &db)

				pub.Publish(smtpStatusRecord(parser.SentStatus, parser.Time{Month: time.December, Day: 2, Hour: 13, Minute: 10, Second: 10}))
				pub.Close()
				<-done

				interval := parseTimeInterval("1999-12-01", "2000-01-03")

				So(db.HasLogs(), ShouldBeTrue)
				So(countByStatus(dashboard, parser.BouncedStatus, interval), ShouldEqual, 0)
				So(countByStatus(dashboard, parser.DeferredStatus, interval), ShouldEqual, 0)
				So(countByStatus(dashboard, parser.SentStatus, interval), ShouldEqual, 1)
			})

			Convey("Insert, reopen, insert", func() {
				func() {
					db, done, pub, _, dtor := buildWs()
					defer dtor()

					initConverter(1999, &db)

					// this one is before the time interval
					pub.Publish(smtpStatusRecord(parser.DeferredStatus, parser.Time{Month: time.November, Day: 2, Hour: 13, Minute: 10, Second: 10}))

					pub.Publish(smtpStatusRecord(parser.SentStatus, parser.Time{Month: time.December, Day: 2, Hour: 13, Minute: 10, Second: 10}))
					pub.Close()
					<-done
				}()

				// reopen workspace and add another log
				db, done, pub, dashboard, dtor := buildWs()
				defer dtor()

				initConverter(1999, &db)

				pub.Publish(smtpStatusRecord(parser.SentStatus, parser.Time{Month: time.December, Day: 4, Hour: 13, Minute: 10, Second: 10}))
				pub.Publish(smtpStatusRecord(parser.DeferredStatus, parser.Time{Month: time.December, Day: 5, Hour: 13, Minute: 10, Second: 10}))
				pub.Publish(noPayloadRecord(parser.Time{Month: time.December, Day: 15, Hour: 13, Minute: 10, Second: 10}))

				pub.Publish(smtpStatusRecord(parser.BouncedStatus, parser.Time{Month: time.March, Day: 10, Hour: 13, Minute: 10, Second: 10}))

				// this one is after the time interval
				pub.Publish(smtpStatusRecord(parser.DeferredStatus, parser.Time{Month: time.April, Day: 2, Hour: 13, Minute: 10, Second: 10}))

				pub.Close()
				<-done

				interval := parseTimeInterval("1999-12-02", "2000-03-11")

				So(db.HasLogs(), ShouldBeTrue)

				So(countByStatus(dashboard, parser.BouncedStatus, interval), ShouldEqual, 1)
				So(countByStatus(dashboard, parser.DeferredStatus, interval), ShouldEqual, 1)
				So(countByStatus(dashboard, parser.SentStatus, interval), ShouldEqual, 2)
			})

			Convey("Many different smtp status", func() {
				db, done, pub, d, dtor := buildWs()
				defer dtor()
				initConverter(1999, &db)

				interval := parseTimeInterval("1999-12-02", "2000-03-11")

				{
					s := parser.SentStatus
					d := parser.DeferredStatus
					b := parser.BouncedStatus

					// Something before the interval
					pub.Publish(smtpStatusRecordWithRecipient(s, t(time.December, 1, 13, 0, 0), "recip", "domain"))

					// Inside the interval
					pub.Publish(smtpStatusRecordWithRecipient(s, t(time.December, 2, 14, 1, 2), "r1", "ALALALA.COM"))
					pub.Publish(smtpStatusRecordWithRecipient(b, t(time.December, 2, 14, 1, 3), "r2", "abcdf.com"))
					pub.Publish(smtpStatusRecordWithRecipient(b, t(time.December, 2, 14, 1, 4), "r3", "alalala.com"))
					pub.Publish(smtpStatusRecordWithRecipient(d, t(time.December, 3, 14, 1, 4), "r3", "EMAIL2.COM"))
					pub.Publish(smtpStatusRecordWithRecipient(d, t(time.December, 5, 15, 1, 0), "r2", "email3.com"))
					pub.Publish(smtpStatusRecordWithRecipient(b, t(time.December, 6, 16, 1, 4), "r3", "ALALALA.COM"))
					pub.Publish(smtpStatusRecordWithRecipient(b, t(time.January, 3, 15, 1, 0), "r2", "abcdf.com"))
					pub.Publish(smtpStatusRecordWithRecipient(d, t(time.January, 4, 15, 1, 0), "r2", "EMAIL1.COM"))
					pub.Publish(smtpStatusRecordWithRecipient(s, t(time.January, 4, 16, 1, 0), "r2", "example1.com"))
					pub.Publish(smtpStatusRecordWithRecipient(s, t(time.January, 4, 16, 2, 1), "r2", "example1.com"))
					pub.Publish(smtpStatusRecordWithRecipient(b, t(time.March, 11, 16, 2, 1), "r100", "email2.com"))

					// Something after the interval
					pub.Publish(smtpStatusRecordWithRecipient(d, t(time.March, 12, 13, 0, 0), "recip", "domain"))
				}

				pub.Close()
				<-done

				Convey("Busiest: used domain, regardless of the status", func() {
					So(topBusiestDomains(d, interval), ShouldResemble, dashboard.Pairs{
						dashboard.Pair{Key: "alalala.com", Value: 3},
						dashboard.Pair{Key: "abcdf.com", Value: 2},
						dashboard.Pair{Key: "email2.com", Value: 2},
						dashboard.Pair{Key: "example1.com", Value: 2},
						dashboard.Pair{Key: "email1.com", Value: 1},
						dashboard.Pair{Key: "email3.com", Value: 1},
					})
				})

				Convey("Bounced: status = bounced", func() {
					So(topBouncedDomains(d, interval), ShouldResemble, dashboard.Pairs{
						dashboard.Pair{Key: "abcdf.com", Value: 2},
						dashboard.Pair{Key: "alalala.com", Value: 2},
						dashboard.Pair{Key: "email2.com", Value: 1},
					})
				})

				Convey("Deferred: status = deferred", func() {
					So(topDeferredDomains(d, interval), ShouldResemble, dashboard.Pairs{
						dashboard.Pair{Key: "email1.com", Value: 1},
						dashboard.Pair{Key: "email2.com", Value: 1},
						dashboard.Pair{Key: "email3.com", Value: 1},
					})
				})

				Convey("Delivery Status", func() {
					So(deliveryStatus(d, interval), ShouldResemble, dashboard.Pairs{
						dashboard.Pair{Key: "sent", Value: 3},
						dashboard.Pair{Key: "bounced", Value: 5},
						dashboard.Pair{Key: "deferred", Value: 3},
					})
				})
			})

			Convey("No inserts, rolling back transaction on timeout (to exercise coverage)", func() {
				_, done, pub, _, dtor := buildWs()
				defer dtor()
				timeToSleep := time.Duration(float32(TransactionTime.Milliseconds())*1.5) * time.Millisecond
				time.Sleep(timeToSleep)
				pub.Close()
				<-done
			})
		})

		parse := func(line string) data.Record {
			h, p, err := parser.Parse([]byte(line))

			if err != nil {
				panic("Parsing line!!!")
			}

			return data.Record{Time: converter.Convert(h.Time), Header: h, Payload: p}
		}

		Convey("Group According to Domain mapping", func() {
			db, done, pub, d, dtor := buildWs()
			defer dtor()

			initConverter(2020, &db)

			{
				s := parser.SentStatus
				d := parser.DeferredStatus
				b := parser.BouncedStatus

				pub.Publish(smtpStatusRecordWithRecipient(d, t(time.January, 1, 1, 0, 0), "p1", "domaintobegrouped.de"))
				pub.Publish(smtpStatusRecordWithRecipient(d, t(time.January, 2, 1, 0, 0), "p1", "another.de"))
				pub.Publish(smtpStatusRecordWithRecipient(d, t(time.January, 2, 2, 0, 0), "p1", "domaintobegrouped.com"))

				pub.Publish(smtpStatusRecordWithRecipient(b, t(time.January, 3, 1, 0, 0), "p1", "domaintobegrouped.de"))
				pub.Publish(smtpStatusRecordWithRecipient(b, t(time.January, 4, 1, 0, 0), "p1", "domaintobegrouped.com"))
				pub.Publish(smtpStatusRecordWithRecipient(b, t(time.January, 5, 1, 0, 0), "p1", "domaintobegrouped.de"))
				pub.Publish(smtpStatusRecordWithRecipient(b, t(time.January, 6, 1, 0, 0), "p1", "another.de"))

				pub.Publish(smtpStatusRecordWithRecipient(s, t(time.January, 6, 1, 0, 0), "p1", "domaintobegrouped.com"))
			}

			pub.Close()
			<-done

			interval := parseTimeInterval(`2020-01-01`, `2020-12-31`)

			So(topBusiestDomains(d, interval), ShouldResemble, dashboard.Pairs{
				dashboard.Pair{Key: "grouped", Value: 6},
				dashboard.Pair{Key: "another.de", Value: 2},
			})

			So(topBouncedDomains(d, interval), ShouldResemble, dashboard.Pairs{
				dashboard.Pair{Key: "grouped", Value: 3},
				dashboard.Pair{Key: "another.de", Value: 1},
			})

			So(topDeferredDomains(d, interval), ShouldResemble, dashboard.Pairs{
				dashboard.Pair{Key: "grouped", Value: 2},
				dashboard.Pair{Key: "another.de", Value: 1},
			})
		})

		Convey("Ignore sending to itself as relay is ignored", func() {
			Convey("Implicit IP", func() {
				// NOTE: it smells like this test (and maybe the others that use a dashboard
				// should be moved to the `dashboard` package, or rely on custom sql queries, independent
				// from the dashboard ones.
				// Right now it feels that the logs database and the dashboard are highly coupled :-(

				db, done, pub, d, dtor := buildWs()
				defer dtor()

				initConverter(2020, &db)

				// this line is ignored, as we noticed that postfix first sends an email to itself, before trying to forward it to the destination
				pub.Publish(parse(`Jun  3 10:40:57 mail postfix/smtp[9710]: 4AA091855DA0: to=<invalid.email@example.com>, relay=127.0.0.1[127.0.0.1]:10024, delay=0.23, delays=0.15/0/0/0.08, dsn=2.0.0, status=sent (250 2.0.0 from MTA(smtp:[127.0.0.1]:10025): 250 2.0.0 Ok: queued as 776E41855DB2)`))

				pub.Publish(parse(`Jun  3 10:40:59 mail postfix/smtp[9890]: 776E41855DB2: to=<invalid.email@example.com>, relay=mx.example.com[11.22.33.44]:25, delay=1.9, delays=0/0/1.5/0.37, dsn=5.1.1, status=bounced (host mx.example.com[11.22.33.44] said: 550 5.1.1 <invalid.email@example.com> User unknown (in reply to RCPT TO command))`))

				pub.Close()
				<-done

				interval := parseTimeInterval(`2020-06-03`, `2020-06-03`)

				So(countByStatus(d, parser.SentStatus, interval), ShouldEqual, 0)
				So(countByStatus(d, parser.BouncedStatus, interval), ShouldEqual, 1)
				So(countByStatus(d, parser.DeferredStatus, interval), ShouldEqual, 0)

				So(deliveryStatus(d, interval), ShouldResemble, dashboard.Pairs{
					dashboard.Pair{Key: "bounced", Value: 1},
				})

				So(topBusiestDomains(d, interval), ShouldResemble, dashboard.Pairs{
					dashboard.Pair{Key: "example.com", Value: 1},
				})

				So(topBouncedDomains(d, interval), ShouldResemble, dashboard.Pairs{
					dashboard.Pair{Key: "example.com", Value: 1},
				})

				So(topDeferredDomains(d, interval), ShouldResemble, dashboard.Pairs{})
			})

			Convey("Explicit IP", func() {
				// NOTE: it smells like this test (and maybe the others that use a dashboard
				// should be moved to the `dashboard` package, or rely on custom sql queries, independent
				// from the dashboard ones.
				// Right now it feels that the logs database and the dashboard are highly coupled :-(

				db, done, pub, d, dtor := buildWs()
				defer dtor()

				initConverter(2020, &db)

				// this line is ignored, as we noticed that postfix first sends an email to itself, before trying to forward it to the destination
				pub.Publish(parse(`Jun  3 10:40:57 mail postfix-1.2.3.4/smtp[9710]: 4AA091855DA0: to=<invalid.email@example.com>, relay=1.2.3.4[1.2.3.4]:10024, delay=0.23, delays=0.15/0/0/0.08, dsn=2.0.0, status=sent (250 2.0.0 from MTA(smtp:[127.0.0.1]:10025): 250 2.0.0 Ok: queued as 776E41855DB2)`))

				pub.Publish(parse(`Jun  3 10:40:59 mail postfix-1.2.3.4/smtp[9890]: 776E41855DB2: to=<invalid.email@example.com>, relay=mx.example.com[11.22.33.44]:25, delay=1.9, delays=0/0/1.5/0.37, dsn=5.1.1, status=bounced (host mx.example.com[11.22.33.44] said: 550 5.1.1 <invalid.email@example.com> User unknown (in reply to RCPT TO command))`))

				pub.Close()
				<-done

				interval := parseTimeInterval(`2020-06-03`, `2020-06-03`)

				So(countByStatus(d, parser.SentStatus, interval), ShouldEqual, 0)
				So(countByStatus(d, parser.BouncedStatus, interval), ShouldEqual, 1)
				So(countByStatus(d, parser.DeferredStatus, interval), ShouldEqual, 0)

				So(deliveryStatus(d, interval), ShouldResemble, dashboard.Pairs{
					dashboard.Pair{Key: "bounced", Value: 1},
				})

				So(topBusiestDomains(d, interval), ShouldResemble, dashboard.Pairs{
					dashboard.Pair{Key: "example.com", Value: 1},
				})

				So(topBouncedDomains(d, interval), ShouldResemble, dashboard.Pairs{
					dashboard.Pair{Key: "example.com", Value: 1},
				})

				So(topDeferredDomains(d, interval), ShouldResemble, dashboard.Pairs{})
			})
		})
	})
}
