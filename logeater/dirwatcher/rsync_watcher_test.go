// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package dirwatcher

import (
	. "github.com/smartystreets/goconvey/convey"
	parser "gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"testing"
	"time"
)

func TestWatchingDirectoryManagedByRsync(t *testing.T) {
	Convey("Keep track of logs contained in a directory updated periodically by rsync", t, func() {
		originDir, clearOriginDir := testutil.TempDir(t)
		dstDir, clearDstDir := testutil.TempDir(t)

		defer clearDstDir()
		defer clearOriginDir()

		type parsedLog struct {
			h parser.Header
			p parser.Payload
		}

		logs := []parsedLog{}

		newRunner := func(filename string, offset int64) rsyncedFileWatcherRunner {
			return newRsyncedFileWatcherRunner(&rsyncedFileWatcher{filename: path.Join(dstDir, filename), offset: offset}, func(h parser.Header, p parser.Payload) {
				logs = append(logs, parsedLog{h: h, p: p})
			})
		}

		syncDir := func() {
			cmd := exec.Command("rsync", "-rav", originDir+"/", dstDir+"/")
			So(cmd.Run(), ShouldBeNil)
		}

		writeFileContent := func(filename string, content string) {
			filePath := path.Join(originDir, filename)
			So(ioutil.WriteFile(filePath, []byte(content), os.ModePerm), ShouldBeNil)
		}

		Convey("Using some offset", func() {
			// Ensure mail.log exists, required by the watcher
			writeFileContent("mail.log", `Jul 19 01:02:03 mail lalala: Useless Payload
Jul 20 01:02:03 mail lalala: Useless Payload`)
			syncDir()

			Convey("No updates", func() {
				w := newRunner("mail.log", 44)

				done, cancel := w.Run()

				cancel()
				done()

				So(logs, ShouldResemble, []parsedLog{
					parsedLog{
						h: parser.Header{
							Time:      parser.Time{Month: time.July, Day: 20, Hour: 1, Minute: 2, Second: 3},
							Host:      "mail",
							Process:   "lalala",
							Daemon:    "",
							PID:       0,
							ProcessIP: nil,
						},
						p: nil,
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

				done, cancel := w.Run()

				cancel()
				done()

				So(logs, ShouldResemble, []parsedLog{
					parsedLog{
						h: parser.Header{
							Time:      parser.Time{Month: time.July, Day: 19, Hour: 1, Minute: 2, Second: 3},
							Host:      "mail",
							Process:   "lalala",
							Daemon:    "",
							PID:       0,
							ProcessIP: nil,
						},
						p: nil,
					},
				})
			})

			Convey("New line is appended and synchronized", func() {
				w := newRunner("mail.log", 0)

				done, cancel := w.Run()

				writeFileContent("mail.log", `Jul 19 01:02:03 mail lalala: Useless Payload
Aug 20 02:03:04 mail cacaca: Useless Payload`)
				syncDir()

				Convey("No other lines are added", func() {
					cancel()
					done()

					So(logs, ShouldResemble, []parsedLog{
						parsedLog{
							h: parser.Header{
								Time:      parser.Time{Month: time.July, Day: 19, Hour: 1, Minute: 2, Second: 3},
								Host:      "mail",
								Process:   "lalala",
								Daemon:    "",
								PID:       0,
								ProcessIP: nil,
							},
							p: nil,
						},
						parsedLog{
							h: parser.Header{
								Time:      parser.Time{Month: time.August, Day: 20, Hour: 2, Minute: 3, Second: 4},
								Host:      "mail",
								Process:   "cacaca",
								Daemon:    "",
								PID:       0,
								ProcessIP: nil,
							},
							p: nil,
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
						parsedLog{
							h: parser.Header{
								Time:      parser.Time{Month: time.July, Day: 19, Hour: 1, Minute: 2, Second: 3},
								Host:      "mail",
								Process:   "lalala",
								Daemon:    "",
								PID:       0,
								ProcessIP: nil,
							},
							p: nil,
						},
						parsedLog{
							h: parser.Header{
								Time:      parser.Time{Month: time.August, Day: 20, Hour: 2, Minute: 3, Second: 4},
								Host:      "mail",
								Process:   "cacaca",
								Daemon:    "",
								PID:       0,
								ProcessIP: nil,
							},
							p: nil,
						},
					})
				})

				Convey("File is rewritten with no content, done by logrotate. No new lines are reported", func() {
					writeFileContent("mail.log", ``)
					syncDir()

					cancel()
					done()

					So(logs, ShouldResemble, []parsedLog{
						parsedLog{
							h: parser.Header{
								Time:      parser.Time{Month: time.July, Day: 19, Hour: 1, Minute: 2, Second: 3},
								Host:      "mail",
								Process:   "lalala",
								Daemon:    "",
								PID:       0,
								ProcessIP: nil,
							},
							p: nil,
						},
						parsedLog{
							h: parser.Header{
								Time:      parser.Time{Month: time.August, Day: 20, Hour: 2, Minute: 3, Second: 4},
								Host:      "mail",
								Process:   "cacaca",
								Daemon:    "",
								PID:       0,
								ProcessIP: nil,
							},
							p: nil,
						},
					})
				})

				Convey("File is rewritten with a new log line, and the file is shorter than before", func() {
					writeFileContent("mail.log", `Aug 21 02:03:04 mail banana: Useless Payload`)
					syncDir()

					cancel()
					done()

					So(logs, ShouldResemble, []parsedLog{
						parsedLog{
							h: parser.Header{
								Time:      parser.Time{Month: time.July, Day: 19, Hour: 1, Minute: 2, Second: 3},
								Host:      "mail",
								Process:   "lalala",
								Daemon:    "",
								PID:       0,
								ProcessIP: nil,
							},
							p: nil,
						},
						parsedLog{
							h: parser.Header{
								Time:      parser.Time{Month: time.August, Day: 20, Hour: 2, Minute: 3, Second: 4},
								Host:      "mail",
								Process:   "cacaca",
								Daemon:    "",
								PID:       0,
								ProcessIP: nil,
							},
							p: nil,
						},
						parsedLog{
							h: parser.Header{
								Time:      parser.Time{Month: time.August, Day: 21, Hour: 2, Minute: 3, Second: 4},
								Host:      "mail",
								Process:   "banana",
								Daemon:    "",
								PID:       0,
								ProcessIP: nil,
							},
							p: nil,
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
						parsedLog{
							h: parser.Header{
								Time:      parser.Time{Month: time.July, Day: 19, Hour: 1, Minute: 2, Second: 3},
								Host:      "mail",
								Process:   "lalala",
								Daemon:    "",
								PID:       0,
								ProcessIP: nil,
							},
							p: nil,
						},
						parsedLog{
							h: parser.Header{
								Time:      parser.Time{Month: time.August, Day: 20, Hour: 2, Minute: 3, Second: 4},
								Host:      "mail",
								Process:   "cacaca",
								Daemon:    "",
								PID:       0,
								ProcessIP: nil,
							},
							p: nil,
						},
						parsedLog{
							h: parser.Header{
								Time:      parser.Time{Month: time.August, Day: 21, Hour: 2, Minute: 3, Second: 4},
								Host:      "mail",
								Process:   "banana",
								Daemon:    "",
								PID:       0,
								ProcessIP: nil,
							},
							p: nil,
						},
						parsedLog{
							h: parser.Header{
								Time:      parser.Time{Month: time.August, Day: 21, Hour: 3, Minute: 3, Second: 4},
								Host:      "mail",
								Process:   "dog",
								Daemon:    "",
								PID:       0,
								ProcessIP: nil,
							},
							p: nil,
						},
						parsedLog{
							h: parser.Header{
								Time:      parser.Time{Month: time.August, Day: 22, Hour: 3, Minute: 3, Second: 4},
								Host:      "mail",
								Process:   "monkey",
								Daemon:    "",
								PID:       0,
								ProcessIP: nil,
							},
							p: nil,
						},
						parsedLog{
							h: parser.Header{
								Time:      parser.Time{Month: time.August, Day: 22, Hour: 4, Minute: 3, Second: 4},
								Host:      "mail",
								Process:   "gorilla",
								Daemon:    "",
								PID:       0,
								ProcessIP: nil,
							},
							p: nil,
						},
						parsedLog{
							h: parser.Header{
								Time:      parser.Time{Month: time.August, Day: 22, Hour: 5, Minute: 3, Second: 4},
								Host:      "mail",
								Process:   "apple",
								Daemon:    "",
								PID:       0,
								ProcessIP: nil,
							},
							p: nil,
						},
					})
				})
			})
		})
	})
}
