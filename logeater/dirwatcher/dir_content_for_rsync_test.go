// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package dirwatcher

import (
	. "github.com/smartystreets/goconvey/convey"
	parsertimeutil "gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser/timeutil"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"path"
	"testing"
	"time"
)

func TestContentForRsync(t *testing.T) {
	Convey("Content for rsync", t, func() {
		originDir, clearOriginDir := testutil.TempDir(t)
		dstDir, clearDstDir := testutil.TempDir(t)

		defer clearDstDir()
		defer clearOriginDir()

		timeFormat, err := parsertimeutil.Get("default")
		So(err, ShouldBeNil)

		syncDir := func() {
			rsyncCommand(originDir, dstDir)
		}

		Convey("File already exist in the directory, no rsync executed", func() {
			writeFileContentWithModificationTime(path.Join(dstDir, "mail.log.1"),
				`Jan  2 01:02:03 mail lalala: Useless Payload`,
				timeutil.MustParseTime(`2000-01-02 02:00:00 +0000`))

			writeFileContentWithModificationTime(path.Join(dstDir, "mail.log"),
				`Jan  3 01:02:03 mail lalala: Useless Payload`,
				timeutil.MustParseTime(`2000-01-03 02:00:00 +0000`))

			content, err := contentForRsyncManagedDirectory(dstDir, timeFormat, DefaultLogPatterns, time.Millisecond*500)
			So(err, ShouldBeNil)

			{
				modTime, err := content.modificationTimeForEntry(path.Join(dstDir, "mail.log.1"))
				So(err, ShouldBeNil)
				So(modTime.In(time.UTC), ShouldResemble, timeutil.MustParseTime(`2000-01-02 02:00:00 +0000`))
			}

			{
				modTime, err := content.modificationTimeForEntry(path.Join(dstDir, "mail.log"))
				So(err, ShouldBeNil)
				So(modTime.In(time.UTC), ShouldResemble, timeutil.MustParseTime(`2000-01-03 02:00:00 +0000`))
			}
		})

		Convey("Files exist only after watching directories start", func() {
			content, err := contentForRsyncManagedDirectory(dstDir, timeFormat, DefaultLogPatterns, time.Millisecond*500)
			So(err, ShouldBeNil)

			writeFileContentWithModificationTime(path.Join(originDir, "mail.log.1"),
				`Jan  2 01:02:03 mail lalala: Useless Payload`,
				timeutil.MustParseTime(`2000-01-02 02:00:00 +0000`))

			writeFileContentWithModificationTime(path.Join(originDir, "mail.log"),
				`Jan  3 01:02:03 mail lalala: Useless Payload`,
				timeutil.MustParseTime(`2000-01-03 02:00:00 +0000`))

			// delay the rsync
			go func() {
				time.Sleep(time.Millisecond * 100)
				syncDir()
			}()

			{
				modTime, err := content.modificationTimeForEntry(path.Join(dstDir, "mail.log.1"))
				So(err, ShouldBeNil)
				So(modTime.In(time.UTC), ShouldResemble, timeutil.MustParseTime(`2000-01-02 02:00:00 +0000`))
			}

			{
				modTime, err := content.modificationTimeForEntry(path.Join(dstDir, "mail.log"))
				So(err, ShouldBeNil)
				So(modTime.In(time.UTC), ShouldResemble, timeutil.MustParseTime(`2000-01-03 02:00:00 +0000`))
			}
		})

		Convey("Files exist only after watching directories start, but with a delay longer than the timeout", func() {
			content, err := contentForRsyncManagedDirectory(dstDir, timeFormat, DefaultLogPatterns, time.Millisecond*500)
			So(err, ShouldBeNil)

			writeFileContentWithModificationTime(path.Join(originDir, "mail.log.1"),
				`Jan  2 01:02:03 mail lalala: Useless Payload`,
				timeutil.MustParseTime(`2000-01-02 02:00:00 +0000`))

			writeFileContentWithModificationTime(path.Join(originDir, "mail.log"),
				`Jan  3 01:02:03 mail lalala: Useless Payload`,
				timeutil.MustParseTime(`2000-01-03 02:00:00 +0000`))

			// delay the rsync
			go func() {
				// 700 is bigger than 500, obviously
				time.Sleep(time.Millisecond * 700)
				syncDir()
			}()

			{
				modTime, err := content.modificationTimeForEntry(path.Join(dstDir, "mail.log.1"))
				So(err, ShouldBeNil)
				So(modTime.In(time.UTC), ShouldResemble, timeutil.MustParseTime(`2000-01-02 02:00:00 +0000`))
			}

			{
				modTime, err := content.modificationTimeForEntry(path.Join(dstDir, "mail.log"))
				So(err, ShouldBeNil)
				So(modTime.In(time.UTC), ShouldResemble, timeutil.MustParseTime(`2000-01-03 02:00:00 +0000`))
			}
		})

		Convey("Incomplete file after watching starts, but later it gets complete", func() {
			content, err := contentForRsyncManagedDirectory(dstDir, timeFormat, DefaultLogPatterns, time.Millisecond*500)
			So(err, ShouldBeNil)

			writeFileContentWithModificationTime(path.Join(originDir, "mail.log.1"),
				`Jan  2 01:02:03 mail lalala: Useless Payload`,
				timeutil.MustParseTime(`2000-01-02 02:00:00 +0000`))

			// delay the rsync
			go func() {
				time.Sleep(time.Millisecond * 100)
				syncDir()

				time.Sleep(time.Millisecond * 600)
				writeFileContentWithModificationTime(path.Join(originDir, "mail.log"),
					`Jan  3 01:02:03 mail lalala: Useless Payload`,
					timeutil.MustParseTime(`2000-01-03 02:00:00 +0000`))
				syncDir()
			}()

			{
				modTime, err := content.modificationTimeForEntry(path.Join(dstDir, "mail.log.1"))
				So(err, ShouldBeNil)
				So(modTime.In(time.UTC), ShouldResemble, timeutil.MustParseTime(`2000-01-02 02:00:00 +0000`))
			}

			{
				modTime, err := content.modificationTimeForEntry(path.Join(dstDir, "mail.log"))
				So(err, ShouldBeNil)
				So(modTime.In(time.UTC), ShouldResemble, timeutil.MustParseTime(`2000-01-03 02:00:00 +0000`))
			}
		})
	})
}
