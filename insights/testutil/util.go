// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package testutil

import (
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
	connPair, removeDir := testutil.TempDBConnectionMigrated(t, "insights")

	creator, err := core.NewCreator(connPair.RwConn)
	if err != nil {
		t.Fatal(err)
	}

	fetcher, err := core.NewFetcher(connPair.RoConnPool)
	if err != nil {
		t.Fatal(err)
	}

	return &FakeAccessor{DBCreator: creator, Fetcher: fetcher, Insights: []int64{}, ConnPair: connPair},
		func() {
			if connPair.Close(); err != nil {
				t.Fatal(err)
			}

			removeDir()
		}
}

func executeCyclesUntil(detector core.Detector, accessor *FakeAccessor, clock *FakeClock, end time.Time, stepDuration time.Duration) error {
	for ; end.After(clock.Time); clock.Sleep(stepDuration) {
		if err := accessor.ConnPair.RwConn.Tx(func(tx *sql.Tx) error {
			return detector.Step(clock, tx)
		}); err != nil {
			return errorutil.Wrap(err)
		}
	}

	return nil
}

func ExecuteCyclesUntil(detector core.Detector, accessor *FakeAccessor, clock *FakeClock, end time.Time, stepDuration time.Duration) {
	So(executeCyclesUntil(detector, accessor, clock, end, stepDuration), ShouldBeNil)
}
