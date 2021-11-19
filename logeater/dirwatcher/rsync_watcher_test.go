// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package dirwatcher

import (
	. "github.com/smartystreets/goconvey/convey"
	parser "gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser"
	parsertimeutil "gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser/timeutil"
	"gitlab.com/lightmeter/controlcenter/pkg/runner"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"testing"
	"time"
)

func rsyncCommand(src, dst string) {
	cmd := exec.Command("rsync", "-rav", src+"/", dst+"/")
	if err := cmd.Start(); err != nil {
		panic(err)
	}

	// give some time to the file rsync watcher to notice the changes. Yes, this is an ugly workaround,
	// to prevent the unit tests to sporadically fail in the gitlab runners, which are quite flaky.
	time.Sleep(time.Millisecond * 500)

	if err := cmd.Wait(); err != nil {
		panic(err)
	}
}

func writeFileContentWithModificationTime(filePath string, content string, modTime time.Time) {
	if err := ioutil.WriteFile(filePath, []byte(content), os.ModePerm); err != nil {
		panic(err)
	}

	if err := os.Chtimes(filePath, modTime, modTime); err != nil {
		panic(err)
	}
}

func TestWatchingDirectoryManagedByRsync(t *testing.T) {
	Convey("Keep track of logs contained in a directory updated periodically by rsync", t, func() {
		originDir, clearOriginDir := testutil.TempDir(t)
		dstDir, clearDstDir := testutil.TempDir(t)

		defer clearDstDir()
		defer clearOriginDir()

		timeFormat, err := parsertimeutil.Get("default")
		So(err, ShouldBeNil)

		type parsedLog struct {
			h parser.Header
		}

		logs := []parsedLog{}

		newRunner := func(filename string, offset int64) rsyncedFileWatcherRunner {
			return newRsyncedFileWatcherRunner(&rsyncedFileWatcher{filename: path.Join(dstDir, filename), offset: offset, format: timeFormat}, func(h parser.Header, _ string, _ int) {
				logs = append(logs, parsedLog{h: h})
			})
		}

		syncDir := func() {
			rsyncCommand(originDir, dstDir)
		}

		writeFileContent := func(filename string, content string) {
			writeFileContentWithModificationTime(path.Join(originDir, filename), content, time.Now())
		}

		Convey("Using some offset", func() {
			// Ensure mail.log exists, required by the watcher
			writeFileContent("mail.log", `Jul 19 01:02:03 mail lalala: Useless Payload
Jul 20 01:02:03 mail lalala: Useless Payload`)
			syncDir()

			Convey("No updates", func() {
				w := newRunner("mail.log", 44)

				done, cancel := runner.Run(w)

				cancel()
				done()

				So(logs, ShouldResemble, []parsedLog{
					{
						h: parser.Header{
							Time:      parser.Time{Month: time.July, Day: 20, Hour: 1, Minute: 2, Second: 3},
							Host:      "mail",
							Process:   "lalala",
							Daemon:    "",
							PID:       0,
							ProcessIP: nil,
						},
					},
				})
			})
		})

		Convey("From offset 0", func() {
			// Ensure mail.log exists, required by the watcher
			writeFileContent("mail.log", `Jul 19 01:02:03 mail lalala: Useless Payload`)
			syncDir()

			Convey("No updates", func() {
				w := newRunner("mail.log", 0)

				done, cancel := runner.Run(w)

				cancel()
				done()

				So(logs, ShouldResemble, []parsedLog{
					{
						h: parser.Header{
							Time:      parser.Time{Month: time.July, Day: 19, Hour: 1, Minute: 2, Second: 3},
							Host:      "mail",
							Process:   "lalala",
							Daemon:    "",
							PID:       0,
							ProcessIP: nil,
						},
					},
				})
			})

			Convey("New line is appended and synchronized", func() {
				w := newRunner("mail.log", 0)

				doneRunning, cancel := runner.Run(w)

				done := func() {
					So(doneRunning(), ShouldBeNil)
				}

				writeFileContent("mail.log", `Jul 19 01:02:03 mail lalala: Useless Payload
Aug 20 02:03:04 mail cacaca: Useless Payload`)
				syncDir()

				Convey("No other lines are added", func() {
					cancel()
					done()

					So(logs, ShouldResemble, []parsedLog{
						{
							h: parser.Header{
								Time:      parser.Time{Month: time.July, Day: 19, Hour: 1, Minute: 2, Second: 3},
								Host:      "mail",
								Process:   "lalala",
								Daemon:    "",
								PID:       0,
								ProcessIP: nil,
							},
						},
						{
							h: parser.Header{
								Time:      parser.Time{Month: time.August, Day: 20, Hour: 2, Minute: 3, Second: 4},
								Host:      "mail",
								Process:   "cacaca",
								Daemon:    "",
								PID:       0,
								ProcessIP: nil,
							},
						},
					})
				})

				Convey("File is rewritten with the same content, nothing changes", func() {
					writeFileContent("mail.log", `Jul 19 01:02:03 mail lalala: Useless Payload
Aug 20 02:03:04 mail cacaca: Useless Payload`)
					syncDir()

					cancel()
					done()

					So(logs, ShouldResemble, []parsedLog{
						{
							h: parser.Header{
								Time:      parser.Time{Month: time.July, Day: 19, Hour: 1, Minute: 2, Second: 3},
								Host:      "mail",
								Process:   "lalala",
								Daemon:    "",
								PID:       0,
								ProcessIP: nil,
							},
						},
						{
							h: parser.Header{
								Time:      parser.Time{Month: time.August, Day: 20, Hour: 2, Minute: 3, Second: 4},
								Host:      "mail",
								Process:   "cacaca",
								Daemon:    "",
								PID:       0,
								ProcessIP: nil,
							},
						},
					})
				})

				Convey("File is rewritten with no content, done by logrotate. No new lines are reported", func() {
					writeFileContent("mail.log", ``)
					syncDir()

					cancel()
					done()

					So(logs, ShouldResemble, []parsedLog{
						{
							h: parser.Header{
								Time:      parser.Time{Month: time.July, Day: 19, Hour: 1, Minute: 2, Second: 3},
								Host:      "mail",
								Process:   "lalala",
								Daemon:    "",
								PID:       0,
								ProcessIP: nil,
							},
						},
						{
							h: parser.Header{
								Time:      parser.Time{Month: time.August, Day: 20, Hour: 2, Minute: 3, Second: 4},
								Host:      "mail",
								Process:   "cacaca",
								Daemon:    "",
								PID:       0,
								ProcessIP: nil,
							},
						},
					})
				})

				Convey("File is rewritten with a new log line, and the file is shorter than before. This line is just appended", func() {
					writeFileContent("mail.log", `Aug 21 02:03:04 mail banana: Useless Payload`)
					syncDir()

					cancel()
					done()

					So(logs, ShouldResemble, []parsedLog{
						{
							h: parser.Header{
								Time:      parser.Time{Month: time.July, Day: 19, Hour: 1, Minute: 2, Second: 3},
								Host:      "mail",
								Process:   "lalala",
								Daemon:    "",
								PID:       0,
								ProcessIP: nil,
							},
						},
						{
							h: parser.Header{
								Time:      parser.Time{Month: time.August, Day: 20, Hour: 2, Minute: 3, Second: 4},
								Host:      "mail",
								Process:   "cacaca",
								Daemon:    "",
								PID:       0,
								ProcessIP: nil,
							},
						},
						{
							h: parser.Header{
								Time:      parser.Time{Month: time.August, Day: 21, Hour: 2, Minute: 3, Second: 4},
								Host:      "mail",
								Process:   "banana",
								Daemon:    "",
								PID:       0,
								ProcessIP: nil,
							},
						},
					})
				})

				Convey("File is rewritten with totally new content, longer than the original one", func() {
					writeFileContent("mail.log", `Aug 21 02:03:04 mail banana: Useless Payload
Aug 21 03:03:04 mail dog: Useless Payload
Aug 22 03:03:04 mail monkey: Useless Payload
Aug 22 04:03:04 mail gorilla: Useless Payload
Aug 22 05:03:04 mail apple: Useless Payload`)
					syncDir()

					cancel()
					done()

					So(logs, ShouldResemble, []parsedLog{
						{
							h: parser.Header{
								Time:      parser.Time{Month: time.July, Day: 19, Hour: 1, Minute: 2, Second: 3},
								Host:      "mail",
								Process:   "lalala",
								Daemon:    "",
								PID:       0,
								ProcessIP: nil,
							},
						},
						{
							h: parser.Header{
								Time:      parser.Time{Month: time.August, Day: 20, Hour: 2, Minute: 3, Second: 4},
								Host:      "mail",
								Process:   "cacaca",
								Daemon:    "",
								PID:       0,
								ProcessIP: nil,
							},
						},
						{
							h: parser.Header{
								Time:      parser.Time{Month: time.August, Day: 21, Hour: 2, Minute: 3, Second: 4},
								Host:      "mail",
								Process:   "banana",
								Daemon:    "",
								PID:       0,
								ProcessIP: nil,
							},
						},
						{
							h: parser.Header{
								Time:      parser.Time{Month: time.August, Day: 21, Hour: 3, Minute: 3, Second: 4},
								Host:      "mail",
								Process:   "dog",
								Daemon:    "",
								PID:       0,
								ProcessIP: nil,
							},
						},
						{
							h: parser.Header{
								Time:      parser.Time{Month: time.August, Day: 22, Hour: 3, Minute: 3, Second: 4},
								Host:      "mail",
								Process:   "monkey",
								Daemon:    "",
								PID:       0,
								ProcessIP: nil,
							},
						},
						{
							h: parser.Header{
								Time:      parser.Time{Month: time.August, Day: 22, Hour: 4, Minute: 3, Second: 4},
								Host:      "mail",
								Process:   "gorilla",
								Daemon:    "",
								PID:       0,
								ProcessIP: nil,
							},
						},
						{
							h: parser.Header{
								Time:      parser.Time{Month: time.August, Day: 22, Hour: 5, Minute: 3, Second: 4},
								Host:      "mail",
								Process:   "apple",
								Daemon:    "",
								PID:       0,
								ProcessIP: nil,
							},
						},
					})
				})
			})
		})
	})
}
