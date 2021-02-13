// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package dbconn

import (
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/util/temputil"
	"path"
	"testing"
)

func init() {
	lmsqlite3.Initialize(lmsqlite3.Options{})
}

func TestCreateManyDatabases(t *testing.T) {
	Convey("Create many databases", t, func() {
		for i := 0; i < 1000; i++ {
			func() {
				dirName, deleteDir := temputil.TempDir(t)
				defer deleteDir()

				filename := path.Join(dirName, "some.db")

				db, err := Open(filename, 10)

				So(err, ShouldBeNil)

				defer func() {
					So(db.Close(), ShouldBeNil)
				}()
			}()
		}
	})
}
