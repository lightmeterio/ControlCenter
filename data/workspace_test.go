package data

import (
	. "github.com/smartystreets/goconvey/convey"
	parser "gitlab.com/lightmeter/postfix-log-parser"
	"io/ioutil"
	"log"
	"os"
	"path"
	"testing"
	"time"
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
			_, err := NewWorkspace("/proc/lalala", Config{Location: time.UTC, DefaultYear: 1999})
			So(err, ShouldNotEqual, nil)
		})

		Convey("Db is a directory instead of a file", func() {
			dir := tempDir()
			log.Println("workspace", dir)
			defer os.RemoveAll(dir)
			So(os.Mkdir(path.Join(dir, "data.db"), os.ModePerm), ShouldEqual, nil)
			_, err := NewWorkspace(dir, Config{Location: time.UTC, DefaultYear: 1999})
			So(err, ShouldNotEqual, nil)
		})

		Convey("Db is not a sqlite file", func() {
			dir := tempDir()
			defer os.RemoveAll(dir)
			ioutil.WriteFile(path.Join(dir, "data.db"), []byte("not a sqlite file header"), os.ModePerm)
			_, err := NewWorkspace(dir, Config{Location: time.UTC, DefaultYear: 1999})
			So(err, ShouldNotEqual, nil)
		})
	})

	Convey("Creation succeeds", t, func() {
		Convey("Create Workspace", func() {
			dir := tempDir()
			defer os.RemoveAll(dir)
			ws, err := NewWorkspace(dir, Config{Location: time.UTC, DefaultYear: 1999})
			So(err, ShouldEqual, nil)

			defer ws.Close()
			So(ws.HasLogs(), ShouldBeFalse)
		})

		Convey("Empty Database is properly closed", func() {
			dir := tempDir()
			defer os.RemoveAll(dir)
			ws, err := NewWorkspace(dir, Config{Location: time.UTC, DefaultYear: 1999})
			So(err, ShouldEqual, nil)
			So(ws.HasLogs(), ShouldBeFalse)
			So(ws.Close(), ShouldEqual, nil)
			So(ws.readerConnection.Stats().OpenConnections, ShouldEqual, 0)
			So(ws.writerConnection.Stats().OpenConnections, ShouldEqual, 0)
		})

		Convey("Reopening workspace succeeds", func() {
			dir := tempDir()
			defer os.RemoveAll(dir)

			ws1, err := NewWorkspace(dir, Config{Location: time.UTC, DefaultYear: 1999})
			ws1.Close()

			ws2, err := NewWorkspace(dir, Config{Location: time.UTC, DefaultYear: 1999})
			So(err, ShouldEqual, nil)
			ws2.Close()
		})
	})

	Convey("Inserting logs", t, func() {
		dir := tempDir()
		defer os.RemoveAll(dir)

		buildWs := func(year int) (Workspace, <-chan interface{}, Publisher, func()) {
			ws, err := NewWorkspace(dir, Config{Location: time.UTC, DefaultYear: year})
			So(err, ShouldEqual, nil)
			done := ws.Run()
			pub := ws.NewPublisher()
			return ws, done, pub, func() { So(ws.Close(), ShouldEqual, nil) }
		}

		smtpStatusRecord := func(status parser.SmtpStatus, t parser.Time) Record {
			return Record{
				Header: parser.Header{
					Time:    t,
					Host:    "mail",
					Process: "smtp",
				},
				Payload: parser.SmtpSentStatus{
					Queue:               "AA",
					RecipientLocalPart:  "recipient",
					RecipientDomainPart: "gmail.com",
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

		noPayloadRecord := func(t parser.Time) Record {
			return Record{
				Header: parser.Header{
					Time:    parser.Time{Day: 3, Month: time.January, Hour: 13, Minute: 15, Second: 45},
					Host:    "mail",
					Process: "smtp",
				},
				Payload: nil,
			}
		}

		parseTimeInterval := func(from, to string) TimeInterval {
			interval, err := ParseTimeInterval(from, to, time.UTC)
			if err != nil {
				panic("pasring interval")
			}
			return interval
		}

		Convey("Inserts nothing", func() {
			ws, done, pub, dtor := buildWs(1999)
			defer dtor()
			dashboard := ws.Dashboard()
			pub.Close()
			<-done

			interval := parseTimeInterval("1999-12-02", "2000-01-03")

			So(ws.HasLogs(), ShouldBeFalse)
			So(dashboard.CountByStatus(parser.BouncedStatus, interval), ShouldEqual, 0)
			So(dashboard.CountByStatus(parser.DeferredStatus, interval), ShouldEqual, 0)
			So(dashboard.CountByStatus(parser.SentStatus, interval), ShouldEqual, 0)
		})

		Convey("Inserts one log entry", func() {
			ws, done, pub, dtor := buildWs(1999)
			defer dtor()
			dashboard := ws.Dashboard()

			pub.Publish(smtpStatusRecord(parser.SentStatus, parser.Time{Month: time.December, Day: 2, Hour: 13, Minute: 10, Second: 10}))
			pub.Close()
			<-done

			interval := parseTimeInterval("1999-12-01", "2000-01-03")

			So(ws.HasLogs(), ShouldBeTrue)
			So(dashboard.CountByStatus(parser.BouncedStatus, interval), ShouldEqual, 0)
			So(dashboard.CountByStatus(parser.DeferredStatus, interval), ShouldEqual, 0)
			So(dashboard.CountByStatus(parser.SentStatus, interval), ShouldEqual, 1)
		})

		Convey("Insert, reopen, insert", func() {
			func() {
				_, done, pub, dtor := buildWs(1999)
				defer dtor()
				// this one is before the time interval
				pub.Publish(smtpStatusRecord(parser.DeferredStatus, parser.Time{Month: time.November, Day: 2, Hour: 13, Minute: 10, Second: 10}))

				pub.Publish(smtpStatusRecord(parser.SentStatus, parser.Time{Month: time.December, Day: 2, Hour: 13, Minute: 10, Second: 10}))
				pub.Close()
				<-done
			}()

			// reopen workspace and add another log
			ws, done, pub, dtor := buildWs(1999)
			defer dtor()

			pub.Publish(smtpStatusRecord(parser.SentStatus, parser.Time{Month: time.December, Day: 4, Hour: 13, Minute: 10, Second: 10}))
			pub.Publish(smtpStatusRecord(parser.DeferredStatus, parser.Time{Month: time.December, Day: 5, Hour: 13, Minute: 10, Second: 10}))
			pub.Publish(noPayloadRecord(parser.Time{Month: time.December, Day: 15, Hour: 13, Minute: 10, Second: 10}))

			pub.Publish(smtpStatusRecord(parser.BouncedStatus, parser.Time{Month: time.March, Day: 10, Hour: 13, Minute: 10, Second: 10}))

			// this one is after the time interval
			pub.Publish(smtpStatusRecord(parser.DeferredStatus, parser.Time{Month: time.April, Day: 2, Hour: 13, Minute: 10, Second: 10}))

			pub.Close()
			<-done

			dashboard := ws.Dashboard()

			interval := parseTimeInterval("1999-12-02", "2000-03-11")

			So(ws.HasLogs(), ShouldBeTrue)

			So(dashboard.CountByStatus(parser.BouncedStatus, interval), ShouldEqual, 1)
			So(dashboard.CountByStatus(parser.DeferredStatus, interval), ShouldEqual, 1)
			So(dashboard.CountByStatus(parser.SentStatus, interval), ShouldEqual, 2)
		})

		Convey("No inserts, rolling back transaction on timeout (to exercise coverage)", func() {
			_, done, pub, dtor := buildWs(1999)
			defer dtor()
			timeToSleep := time.Duration(float32(sqliteTransactionTime.Milliseconds())*1.5) * time.Millisecond
			time.Sleep(timeToSleep)
			pub.Close()
			<-done
		})
	})
}
