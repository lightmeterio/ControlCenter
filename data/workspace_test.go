package data

import (
	. "github.com/smartystreets/goconvey/convey"
	parser "gitlab.com/lightmeter/postfix-log-parser"
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"
)

func TestWorkspaceCreation(t *testing.T) {
	Convey("Creation fails on several scenarios", t, func() {
		Convey("No Permission on workspace", func() {
			// FIXME: this is relying on linux properties, as /proc is a read-only directory
			_, err := NewWorkspace("/proc/lalala", Config{Location: time.UTC, DefaultYear: 1999})
			So(err, ShouldNotEqual, nil)
		})

		Convey("Db is a directory instead of a file", func() {
			dir, _ := ioutil.TempDir("", "")
			defer os.RemoveAll(dir)
			So(os.Mkdir(path.Join(dir, "data.db"), os.ModePerm), ShouldEqual, nil)
			_, err := NewWorkspace(dir, Config{Location: time.UTC, DefaultYear: 1999})
			So(err, ShouldNotEqual, nil)
		})

		Convey("Db is not a sqlite file", func() {
			dir, _ := ioutil.TempDir("", "")
			defer os.RemoveAll(dir)
			ioutil.WriteFile(path.Join(dir, "data.db"), []byte("not a sqlite file header"), os.ModePerm)
			_, err := NewWorkspace(dir, Config{Location: time.UTC, DefaultYear: 1999})
			So(err, ShouldNotEqual, nil)
		})
	})

	Convey("Creation succeeds", t, func() {
		Convey("Create Workspace", func() {
			dir, _ := ioutil.TempDir("", "")
			defer os.RemoveAll(dir)
			ws, err := NewWorkspace(dir, Config{Location: time.UTC, DefaultYear: 1999})
			defer ws.Close()
			So(ws.HasLogs(), ShouldBeFalse)
			So(err, ShouldEqual, nil)
		})

		Convey("Empty Database is properly closed", func() {
			dir, _ := ioutil.TempDir("", "")
			defer os.RemoveAll(dir)
			ws, err := NewWorkspace(dir, Config{Location: time.UTC, DefaultYear: 1999})
			So(ws.HasLogs(), ShouldBeFalse)
			So(err, ShouldEqual, nil)
			So(ws.Close(), ShouldEqual, nil)
			So(ws.readerConnection.Stats().OpenConnections, ShouldEqual, 0)
			So(ws.writerConnection.Stats().OpenConnections, ShouldEqual, 0)
		})

		Convey("Reopening workspace succeeds", func() {
			dir, _ := ioutil.TempDir("", "")
			defer os.RemoveAll(dir)

			ws1, err := NewWorkspace(dir, Config{Location: time.UTC, DefaultYear: 1999})
			ws1.Close()

			ws2, err := NewWorkspace(dir, Config{Location: time.UTC, DefaultYear: 1999})
			So(err, ShouldEqual, nil)
			ws2.Close()
		})
	})

	Convey("Inserting logs", t, func() {
		dir, _ := ioutil.TempDir("", "")
		defer os.RemoveAll(dir)

		buildWs := func() (Workspace, <-chan interface{}, Publisher, func()) {
			ws, err := NewWorkspace(dir, Config{Location: time.UTC, DefaultYear: 1999})
			So(err, ShouldEqual, nil)
			done := ws.Run()
			pub := ws.NewPublisher()
			return ws, done, pub, func() { So(ws.Close(), ShouldEqual, nil) }
		}

		// something done in December
		smtpSentStatusSentRecord := Record{
			Header: parser.Header{
				Time:    parser.Time{Day: 3, Month: time.December, Hour: 13, Minute: 15, Second: 45},
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
				Status:              parser.SentStatus,
				ExtraMessage:        "",
			},
		}

		// this one will force the year to be bumped, as it's in January next year
		smtpDeferredStatusSentRecord := Record{
			Header: parser.Header{
				Time:    parser.Time{Day: 3, Month: time.January, Hour: 13, Minute: 15, Second: 45},
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
				Status:              parser.DeferredStatus,
				ExtraMessage:        "",
			},
		}

		noPayloadRecord := Record{
			Header: parser.Header{
				Time:    parser.Time{Day: 3, Month: time.January, Hour: 13, Minute: 15, Second: 45},
				Host:    "mail",
				Process: "smtp",
			},
			Payload: nil,
		}

		Convey("Inserts nothing", func() {
			ws, done, pub, dtor := buildWs()
			defer dtor()
			dashboard := ws.Dashboard()
			pub.Close()
			<-done

			So(ws.HasLogs(), ShouldBeFalse)
			So(dashboard.CountByStatus(parser.BouncedStatus), ShouldEqual, 0)
			So(dashboard.CountByStatus(parser.DeferredStatus), ShouldEqual, 0)
			So(dashboard.CountByStatus(parser.SentStatus), ShouldEqual, 0)
		})

		Convey("Inserts one log entry", func() {
			ws, done, pub, dtor := buildWs()
			defer dtor()
			dashboard := ws.Dashboard()

			pub.Publish(smtpSentStatusSentRecord)
			pub.Close()
			<-done

			So(ws.HasLogs(), ShouldBeTrue)
			So(dashboard.CountByStatus(parser.BouncedStatus), ShouldEqual, 0)
			So(dashboard.CountByStatus(parser.DeferredStatus), ShouldEqual, 0)
			So(dashboard.CountByStatus(parser.SentStatus), ShouldEqual, 1)
		})

		Convey("Insert, reopen, insert", func() {
			func() {
				_, done, pub, dtor := buildWs()
				defer dtor()
				pub.Publish(smtpSentStatusSentRecord)
				pub.Close()
				<-done
			}()

			// reopen workspace and add another log
			ws, done, pub, dtor := buildWs()
			defer dtor()

			pub.Publish(smtpDeferredStatusSentRecord)
			pub.Publish(smtpSentStatusSentRecord)
			pub.Publish(noPayloadRecord)

			pub.Close()
			<-done

			dashboard := ws.Dashboard()

			So(ws.HasLogs(), ShouldBeTrue)

			So(dashboard.CountByStatus(parser.BouncedStatus), ShouldEqual, 0)
			So(dashboard.CountByStatus(parser.DeferredStatus), ShouldEqual, 1)
			So(dashboard.CountByStatus(parser.SentStatus), ShouldEqual, 2)
		})

		Convey("No inserts, rolling back transaction on timeout (to exercise coverage)", func() {
			_, done, pub, dtor := buildWs()
			defer dtor()
			timeToSleep := time.Duration(float32(sqliteTransactionTime.Milliseconds())*1.5) * time.Millisecond
			time.Sleep(timeToSleep)
			pub.Close()
			<-done
		})
	})
}
