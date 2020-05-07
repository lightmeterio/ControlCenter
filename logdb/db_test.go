package logdb

import (
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/dashboard"
	"gitlab.com/lightmeter/controlcenter/data"
	parser "gitlab.com/lightmeter/postfix-log-parser"
)

func tempDir() string {
	dir, e := ioutil.TempDir("", "lightmeter-tests-*")
	if e != nil {
		panic("error creating temp dir")
	}
	return dir
}

func TestWorkspaceCreation(t *testing.T) {
	Convey("Creation fails on several scenarios", t, func() {
		Convey("No Permission on workspace", func() {
			// FIXME: this is relying on linux properties, as /proc is a read-only directory
			_, err := Open("/proc/lalala", data.Config{Location: time.UTC, DefaultYear: 1999})
			So(err, ShouldNotEqual, nil)
		})

		Convey("Db is a directory instead of a file", func() {
			dir := tempDir()
			defer os.RemoveAll(dir)
			So(os.Mkdir(path.Join(dir, "logs.db"), os.ModePerm), ShouldEqual, nil)
			_, err := Open(dir, data.Config{Location: time.UTC, DefaultYear: 1999})
			So(err, ShouldNotEqual, nil)
		})

		Convey("Db is not a sqlite file", func() {
			dir := tempDir()
			defer os.RemoveAll(dir)
			ioutil.WriteFile(path.Join(dir, "logs.db"), []byte("not a sqlite file header"), os.ModePerm)
			_, err := Open(dir, data.Config{Location: time.UTC, DefaultYear: 1999})
			So(err, ShouldNotEqual, nil)
		})
	})

	Convey("Creation succeeds", t, func() {
		Convey("Create DB", func() {
			dir := tempDir()
			defer os.RemoveAll(dir)
			db, err := Open(dir, data.Config{Location: time.UTC, DefaultYear: 1999})
			So(err, ShouldEqual, nil)

			defer db.Close()
			So(db.HasLogs(), ShouldBeFalse)
		})

		Convey("Empty Database is properly closed", func() {
			dir := tempDir()
			defer os.RemoveAll(dir)
			db, err := Open(dir, data.Config{Location: time.UTC, DefaultYear: 1999})
			So(err, ShouldEqual, nil)
			So(db.HasLogs(), ShouldBeFalse)
			So(db.Close(), ShouldEqual, nil)
		})

		Convey("Reopening workspace succeeds", func() {
			dir := tempDir()
			defer os.RemoveAll(dir)

			ws1, err := Open(dir, data.Config{Location: time.UTC, DefaultYear: 1999})
			ws1.Close()

			ws2, err := Open(dir, data.Config{Location: time.UTC, DefaultYear: 1999})
			So(err, ShouldEqual, nil)
			ws2.Close()
		})
	})

	Convey("Inserting logs", t, func() {
		dir := tempDir()
		defer os.RemoveAll(dir)

		buildWs := func(year int) (DB, <-chan interface{}, data.Publisher, dashboard.Dashboard, func()) {
			db, err := Open(dir, data.Config{Location: time.UTC, DefaultYear: year})
			So(err, ShouldEqual, nil)
			done := db.Run()
			pub := db.NewPublisher()
			dashboard, err := dashboard.New(db.ReadConnection())
			So(err, ShouldEqual, nil)
			return db, done, pub, dashboard, func() {
				So(dashboard.Close(), ShouldEqual, nil)
				So(db.Close(), ShouldEqual, nil)
			}
		}

		smtpStatusRecordWithRecipient := func(status parser.SmtpStatus, t parser.Time, recipientLocalPart, recipientDomainPart string) data.Record {
			return data.Record{
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
			return data.Record{
				Header: parser.Header{
					Time:    parser.Time{Day: 3, Month: time.January, Hour: 13, Minute: 15, Second: 45},
					Host:    "mail",
					Process: "smtp",
				},
				Payload: nil,
			}
		}

		parseTimeInterval := func(from, to string) data.TimeInterval {
			interval, err := data.ParseTimeInterval(from, to, time.UTC)
			if err != nil {
				panic("pasring interval")
			}
			return interval
		}

		Convey("Inserts nothing", func() {
			db, done, pub, dashboard, dtor := buildWs(1999)
			defer dtor()
			pub.Close()
			<-done

			interval := parseTimeInterval("1999-12-02", "2000-01-03")

			So(db.HasLogs(), ShouldBeFalse)
			So(dashboard.CountByStatus(parser.BouncedStatus, interval), ShouldEqual, 0)
			So(dashboard.CountByStatus(parser.DeferredStatus, interval), ShouldEqual, 0)
			So(dashboard.CountByStatus(parser.SentStatus, interval), ShouldEqual, 0)
		})

		Convey("Inserts one log entry", func() {
			db, done, pub, dashboard, dtor := buildWs(1999)
			defer dtor()

			pub.Publish(smtpStatusRecord(parser.SentStatus, parser.Time{Month: time.December, Day: 2, Hour: 13, Minute: 10, Second: 10}))
			pub.Close()
			<-done

			interval := parseTimeInterval("1999-12-01", "2000-01-03")

			So(db.HasLogs(), ShouldBeTrue)
			So(dashboard.CountByStatus(parser.BouncedStatus, interval), ShouldEqual, 0)
			So(dashboard.CountByStatus(parser.DeferredStatus, interval), ShouldEqual, 0)
			So(dashboard.CountByStatus(parser.SentStatus, interval), ShouldEqual, 1)
		})

		Convey("Insert, reopen, insert", func() {
			func() {
				_, done, pub, _, dtor := buildWs(1999)
				defer dtor()
				// this one is before the time interval
				pub.Publish(smtpStatusRecord(parser.DeferredStatus, parser.Time{Month: time.November, Day: 2, Hour: 13, Minute: 10, Second: 10}))

				pub.Publish(smtpStatusRecord(parser.SentStatus, parser.Time{Month: time.December, Day: 2, Hour: 13, Minute: 10, Second: 10}))
				pub.Close()
				<-done
			}()

			// reopen workspace and add another log
			db, done, pub, dashboard, dtor := buildWs(1999)
			defer dtor()

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

			So(dashboard.CountByStatus(parser.BouncedStatus, interval), ShouldEqual, 1)
			So(dashboard.CountByStatus(parser.DeferredStatus, interval), ShouldEqual, 1)
			So(dashboard.CountByStatus(parser.SentStatus, interval), ShouldEqual, 2)
		})

		Convey("Many different smtp status", func() {
			_, done, pub, d, dtor := buildWs(1999)
			defer dtor()

			interval := parseTimeInterval("1999-12-02", "2000-03-11")

			t := func(mo time.Month, d, h, m, s uint8) parser.Time {
				return parser.Time{Month: mo, Day: d, Hour: h, Minute: m, Second: s}
			}

			{
				s := parser.SentStatus
				d := parser.DeferredStatus
				b := parser.BouncedStatus

				// Something before the interval
				pub.Publish(smtpStatusRecordWithRecipient(s, t(time.December, 1, 13, 0, 0), "recip", "domain"))

				// Inside the interval
				pub.Publish(smtpStatusRecordWithRecipient(s, t(time.December, 2, 14, 1, 2), "r1", "alalala.com"))
				pub.Publish(smtpStatusRecordWithRecipient(b, t(time.December, 2, 14, 1, 3), "r2", "abcdf.com"))
				pub.Publish(smtpStatusRecordWithRecipient(b, t(time.December, 2, 14, 1, 4), "r3", "alalala.com"))
				pub.Publish(smtpStatusRecordWithRecipient(d, t(time.December, 3, 14, 1, 4), "r3", "email2.com"))
				pub.Publish(smtpStatusRecordWithRecipient(d, t(time.December, 5, 15, 1, 0), "r2", "email3.com"))
				pub.Publish(smtpStatusRecordWithRecipient(b, t(time.December, 6, 16, 1, 4), "r3", "alalala.com"))
				pub.Publish(smtpStatusRecordWithRecipient(b, t(time.January, 3, 15, 1, 0), "r2", "abcdf.com"))
				pub.Publish(smtpStatusRecordWithRecipient(d, t(time.January, 4, 15, 1, 0), "r2", "email1.com"))
				pub.Publish(smtpStatusRecordWithRecipient(s, t(time.January, 4, 16, 1, 0), "r2", "example1.com"))
				pub.Publish(smtpStatusRecordWithRecipient(s, t(time.January, 4, 16, 2, 1), "r2", "example1.com"))
				pub.Publish(smtpStatusRecordWithRecipient(b, t(time.March, 11, 16, 2, 1), "r100", "email2.com"))

				// Something after the interval
				pub.Publish(smtpStatusRecordWithRecipient(d, t(time.March, 12, 13, 0, 0), "recip", "domain"))
			}

			pub.Close()
			<-done

			Convey("Busiest: used domain, regardless of the status", func() {
				So(d.TopBusiestDomains(interval), ShouldResemble, dashboard.Pairs{
					dashboard.Pair{Key: "alalala.com", Value: 3},
					dashboard.Pair{Key: "abcdf.com", Value: 2},
					dashboard.Pair{Key: "email2.com", Value: 2},
					dashboard.Pair{Key: "example1.com", Value: 2},
					dashboard.Pair{Key: "email1.com", Value: 1},
					dashboard.Pair{Key: "email3.com", Value: 1},
				})
			})

			Convey("Bounced: status = bounced", func() {
				So(d.TopBouncedDomains(interval), ShouldResemble, dashboard.Pairs{
					dashboard.Pair{Key: "abcdf.com", Value: 2},
					dashboard.Pair{Key: "alalala.com", Value: 2},
					dashboard.Pair{Key: "email2.com", Value: 1},
				})
			})

			Convey("Deferred: status = deferred", func() {
				So(d.TopDeferredDomains(interval), ShouldResemble, dashboard.Pairs{
					dashboard.Pair{Key: "email1.com", Value: 1},
					dashboard.Pair{Key: "email2.com", Value: 1},
					dashboard.Pair{Key: "email3.com", Value: 1},
				})
			})

			Convey("Delivery Status", func() {
				So(d.DeliveryStatus(interval), ShouldResemble, dashboard.Pairs{
					dashboard.Pair{Key: "sent", Value: 3},
					dashboard.Pair{Key: "bounced", Value: 5},
					dashboard.Pair{Key: "deferred", Value: 3},
				})
			})
		})

		Convey("No inserts, rolling back transaction on timeout (to exercise coverage)", func() {
			_, done, pub, _, dtor := buildWs(1999)
			defer dtor()
			timeToSleep := time.Duration(float32(TransactionTime.Milliseconds())*1.5) * time.Millisecond
			time.Sleep(timeToSleep)
			pub.Close()
			<-done
		})
	})
}
