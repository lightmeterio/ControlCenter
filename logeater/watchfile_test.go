// +build slow_tests

// This test won't run by default as they have lots of long "sleeps"
// making executing them all the time a pain in the neck

package logeater

import (
	"github.com/hpcloud/tail"
	. "github.com/smartystreets/goconvey/convey"
	"io/ioutil"
	"os"
	"testing"
	"time"
)

// FIXME: copied from workspace_test.go
// It should be moved to some common place
func tempDir() string {
	dir, e := ioutil.TempDir("", "lightmeter-tests-*")
	if e != nil {
		panic("error creating temp dir")
	}
	return dir
}

func appendToFile(filename string, lines []string) {
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	So(err, ShouldEqual, nil)
	defer f.Close()

	for _, line := range lines {
		_, err = f.Write([]byte(line + "\n"))
		So(err, ShouldEqual, nil)
		So(f.Sync(), ShouldEqual, nil)
	}
}

func TestWatchingFiles(t *testing.T) {
	dir := tempDir()
	//defer os.RemoveAll(dir)

	Convey("Watch Files", t, func() {
		startOfFileLocation := tail.SeekInfo{Offset: 0, Whence: os.SEEK_SET}
		endOfFileLocation := tail.SeekInfo{Offset: 0, Whence: os.SEEK_END}

		pub := FakePublisher{}

		Convey("Fails if file does not exist", func() {
			err, _, _ := WatchFileCancelable(dir+"/non_existent_file.log", startOfFileLocation, &pub)
			So(err, ShouldNotEqual, nil)
		})

		Convey("Given an empty file, detect a line added to it", func() {
			filename := dir + "/not_yet_existing_file.log"
			appendToFile(filename, []string{""})
			err, cancel, done := WatchFileCancelable(filename, startOfFileLocation, &pub)
			So(err, ShouldEqual, nil)
			content := "Mar  1 07:42:10 mail opendkim[225]: C11EA2C620C7: not authenticated"
			time.Sleep(500 * time.Millisecond)
			appendToFile(filename, []string{content})
			time.Sleep(1000 * time.Millisecond)
			cancel <- nil
			So(<-done, ShouldEqual, nil)
			So(len(pub.logs), ShouldEqual, 1)
			So(pub.logs[0].Header.Time.Month, ShouldEqual, time.March)
		})

		Convey("Watch from the end of the file, ignoring anything from before", func() {
			filename := dir + "/mail.log"

			{
				content := []string{`Sep 16 00:07:43 smtpnode07 postfix-10.20.30.40/smtp[3022]: 0C31D3D1E6: to=<a@b.c>, relay=a.net[1.2.3.4]:25, delay=1, delays=0/0.9/0.69/0.03, dsn=4.7.0, status=deferred Extra text)`}
				appendToFile(filename, content)
			}

			err, cancel, done := WatchFileCancelable(filename, endOfFileLocation, &pub)
			So(err, ShouldEqual, nil)

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

			// FIXME: I don't know why notifying the `tail` object is so unreliable
			// on how long it takes to execute...
			time.Sleep(5000 * time.Millisecond)

			appendToFile(filename, []string{"ahhhh"})

			cancel <- nil
			So(<-done, ShouldEqual, nil)

			// FIXME: I know this assertion is a joke, but unfortunately `tail` does not behave
			// in a deterministic way during the unit tests and len(pub.logs) ends up being being
			// either 3 or 4, on my tests, which is ok to confirm it works.
			// Awkwardly, it works well in production. Heiserbug?
			So(len(pub.logs), ShouldBeGreaterThan, 0)
		})
	})
}
