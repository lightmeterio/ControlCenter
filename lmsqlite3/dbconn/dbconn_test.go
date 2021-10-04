// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package dbconn

import (
	"context"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/util/temputil"
	"path"
	"testing"
	"time"
)

func init() {
	lmsqlite3.Initialize(lmsqlite3.Options{})
}

func TestAcquire(t *testing.T) {
	Convey("Acquiring read only connection can be cancelled, if it takes too long", t, func() {
		dir, removeDir := temputil.TempDir(t)
		defer removeDir()

		const poolSize = 2

		db, err := Open(path.Join(dir, "database.db"), poolSize)
		So(err, ShouldBeNil)

		defer func() {
			So(db.Close(), ShouldBeNil)
		}()

		// Important reminder to the reader:
		// In go, deferring code makes it be called at the end of the
		// function where it's been called, **NOT** at the end of a given scope.
		// In this test, we use scopes {} to reuse variable names, and anonymous
		// functions func(){}() to force deferred code to be called

		{
			// acquiring the first connection succeeds
			ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
			defer cancel()
			_, release, err := db.RoConnPool.AcquireContext(ctx)
			So(err, ShouldBeNil)
			defer release()
		}

		func() {
			{
				// acquiring the second connection succeeds
				ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
				defer cancel()
				_, release, err := db.RoConnPool.AcquireContext(ctx)
				So(err, ShouldBeNil)
				defer release()
			}

			{
				// as the two connections in the pool are already taken, trying to acquire another one
				// will fail, due timeout
				ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
				defer cancel()
				_, _, err := db.RoConnPool.AcquireContext(ctx)
				So(err, ShouldNotBeNil)
			}
		}()

		{
			// Here, the second connection has been released, so we'll succeed in obtaining it
			ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
			defer cancel()
			_, release, err := db.RoConnPool.AcquireContext(ctx)
			So(err, ShouldBeNil)
			defer release()
		}
	})
}
