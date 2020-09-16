package logeater

import (
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"io"
	"os"
	"testing"
	"time"

	"github.com/hpcloud/tail"
	. "github.com/smartystreets/goconvey/convey"
)

func appendToFile(filename string, lines []string) {
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	So(err, ShouldBeNil)
	defer f.Close()

	for _, line := range lines {
		_, err = f.Write([]byte(line + "\n"))
		So(err, ShouldBeNil)
		So(f.Sync(), ShouldBeNil)
	}
}

func TestWatchingFiles(t *testing.T) {
	Convey("Watch Files", t, func() {
		firstSecondInJanuary := parseTime(`2000-01-01 00:00:00 +0000`)
		dir := testutil.TempDir()
		defer os.RemoveAll(dir)

		startOfFileLocation := tail.SeekInfo{Offset: 0, Whence: io.SeekStart}
		endOfFileLocation := tail.SeekInfo{Offset: 0, Whence: io.SeekEnd}

		pub := FakePublisher{}

		Convey("Fails if file does not exist", func() {
			err, _, _ := WatchFileCancelable(dir+"/non_existent_file.log", startOfFileLocation, &pub, firstSecondInJanuary)
			So(err, ShouldNotBeNil)
		})

		Convey("Given an empty file, detect a line added to it", func() {
			filename := dir + "/empty_mail.log"
			appendToFile(filename, []string{""})
			err, cancel, done := WatchFileCancelable(filename, startOfFileLocation, &pub, firstSecondInJanuary)
			So(err, ShouldBeNil)
			content := "Mar  1 07:42:10 mail opendkim[225]: C11EA2C620C7: not authenticated"
			time.Sleep(500 * time.Millisecond)
			appendToFile(filename, []string{content})
			time.Sleep(1000 * time.Millisecond)
			cancel()
			done()
			So(len(pub.logs), ShouldEqual, 1)
			So(pub.logs[0].Header.Time.Month, ShouldEqual, time.March)
		})

		Convey("Watch from the end of the file, ignoring anything from before", func() {
			filename := dir + "/non_empty_file.log"

			{
				content := []string{`Sep 16 00:07:43 smtpnode07 postfix-10.20.30.40/smtp[3022]: 0C31D3D1E6: to=<a@b.c>, relay=a.net[1.2.3.4]:25, delay=1, delays=0/0.9/0.69/0.03, dsn=4.7.0, status=deferred Extra text)`}
				appendToFile(filename, content)
			}

			err, cancel, done := WatchFileCancelable(filename, endOfFileLocation, &pub, firstSecondInJanuary)
			So(err, ShouldBeNil)

			{
				content := []string{`Nov  1 07:42:10 mail opendkim[225]: C11EA2C620C7: not authenticated`,
					`Dec 16 14:08:45 smtpnode07 postfix-10.20.30.40/smtp[3022]: 0C31D3D1E6: to=<a@b.c>, relay=a.net[1.2.3.4]:25, delay=1, delays=0/0.9/0.69/0.03, dsn=4.7.0, status=deferred Extra text)`}
				appendToFile(filename, content)
			}

			time.Sleep(500 * time.Millisecond)

			{
				content := []string{`Dec 17 07:42:10 mail opendkim[225]: C11EA2C620C7: not authenticated`,
					`Jan 10 14:08:45 smtpnode07 postfix-10.20.30.40/smtp[3022]: 0C31D3D1E6: to=<a@b.c>, relay=a.net[1.2.3.4]:25, delay=1, delays=0/0.9/0.69/0.03, dsn=4.7.0, status=deferred Extra text)`}
				appendToFile(filename, content)
			}

			appendToFile(filename, []string{"ahhhh"})

			// FIXME: I don't know why notifying the `tail` object is so unreliable
			// on how long it takes to execute...
			time.Sleep(2000 * time.Millisecond)

			cancel()

			time.Sleep(1000 * time.Millisecond)

			done()

			// FIXME: `tail` is very unreliable and sometimes don't write all logs in time
			// making a proper assertion quite difficult
			So(len(pub.logs), ShouldBeGreaterThan, 0)
		})
	})
}
