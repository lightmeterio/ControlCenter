// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package testutil

import (
	// required by the data migrator
	"context"
	"database/sql"
	//nolint:golint
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/insights/core"
	_ "gitlab.com/lightmeter/controlcenter/insights/migrations"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"testing"
	"time"
)

type FakeClock = timeutil.FakeClock

type FakeAccessor struct {
	*core.DBCreator
	core.Fetcher
	Insights []int64
	ConnPair *dbconn.PooledPair
}

func (c *FakeAccessor) GenerateInsight(ctx context.Context, tx *sql.Tx, properties core.InsightProperties) error {
	id, err := core.GenerateInsight(ctx, tx, properties)

	if err != nil {
		return errorutil.Wrap(err)
	}

	c.Insights = append(c.Insights, id)

	return nil
}

// NewFakeAccessor returns an acessor that implements core.Fetcher and core.Creator
// using a temporary database that should be delete using clear()
func NewFakeAccessor(t *testing.T) (acessor *FakeAccessor, clear func()) {
	_, closeDatabases := testutil.TempDatabases(t)

	creator, err := core.NewCreator(dbconn.Db("insights").RwConn)
	if err != nil {
		t.Fatal(err)
	}

	fetcher, err := core.NewFetcher(dbconn.Db("insights").RoConnPool)
	if err != nil {
		t.Fatal(err)
	}

	return &FakeAccessor{DBCreator: creator, Fetcher: fetcher, Insights: []int64{}, ConnPair: dbconn.Db("insights")},
		func() {
			if err := dbconn.Db("insights").Close(); err != nil {
				t.Fatal(err)
			}

			closeDatabases()
		}
}

func executeCyclesUntil(detector core.Detector, accessor *FakeAccessor, clock *FakeClock, end time.Time, stepDuration time.Duration) error {
	for ; end.After(clock.Time); clock.Sleep(stepDuration) {
		tx, err := accessor.ConnPair.RwConn.Begin()
		if err != nil {
			return errorutil.Wrap(err)
		}

		if err := detector.Step(clock, tx); err != nil {
			return errorutil.Wrap(err)
		}

		if err := tx.Commit(); err != nil {
			return errorutil.Wrap(err)
		}
	}

	return nil
}

func ExecuteCyclesUntil(detector core.Detector, accessor *FakeAccessor, clock *FakeClock, end time.Time, stepDuration time.Duration) {
	So(executeCyclesUntil(detector, accessor, clock, end, stepDuration), ShouldBeNil)
}
