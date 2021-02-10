// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package migrations

import (
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/insights/core"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	_ "gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/migrator"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"testing"
)

func init() {
	lmsqlite3.Initialize(lmsqlite3.Options{})
}

type fakeContent struct {
	// intentionally CamelCase.
	// the migration `2` should make it snake_case
	From string `json:"From"`
}

func (c fakeContent) String() string {
	return ""
}

func (c fakeContent) Args() []interface{} {
	return nil
}

func (c fakeContent) TplString() string {
	return ""
}

func init() {
	core.RegisterContentType("fake_content_type", 999, core.DefaultContentTypeDecoder(&fakeContent{}))
}

func TestDatabaseMigrationUp(t *testing.T) {
	Convey("Migration succeeds", t, func() {
		connPair, clear := testutil.TempDBConnection(t, "insights.db")
		defer clear()

		Convey("Test json names fixup", func() {
			// Initial setup
			err := migrator.Run(connPair.RwConn.DB, "insights")
			So(err, ShouldBeNil)

			// Then migrate back to version 1, before fixing the json values
			err = migrator.DownTo(connPair.RwConn.DB, 1, "insights")
			So(err, ShouldBeNil)

			{
				tx, err := connPair.RwConn.Begin()
				So(err, ShouldBeNil)

				fakeContent := &fakeContent{
					From: "from",
				}

				_, err = core.GenerateInsight(tx, core.InsightProperties{
					Time:        testutil.MustParseTime(`2006-01-02 15:04:05 -0700`),
					Category:    core.ComparativeCategory,
					ContentType: `fake_content_type`,
					Rating:      core.GoodRating,
					Content:     fakeContent,
				})

				So(err, ShouldBeNil)

				So(tx.Commit(), ShouldBeNil)
			}

			// Then migrate up again
			err = migrator.Run(connPair.RwConn.DB, "insights")
			So(err, ShouldBeNil)

			conn, release := connPair.RoConnPool.Acquire()

			defer release()

			var content string
			err = conn.QueryRow("select content from insights where rowid = ?", 1).Scan(&content)
			So(err, ShouldBeNil)

			// From, CamelCase has been updated to from, snake_case
			So(content, ShouldEqual, `{"from":"from"}`)
		})
	})
}
