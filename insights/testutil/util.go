// SPDX-FileCopyrightText: 2020,  Lightmeter <hello@lightmeter.io>
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0

package testutil

import (
	// required by the data migrator
	"database/sql"
	"gitlab.com/lightmeter/controlcenter/insights/core"
	_ "gitlab.com/lightmeter/controlcenter/insights/migrations"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/migrator"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"sync"
	"testing"
	"time"
)

type FakeClock struct {
	// Locking used just to prevent the race detector of triggering errors during tests
	sync.Mutex
	time.Time
}

func (t *FakeClock) Now() time.Time {
	t.Lock()
	defer t.Unlock()

	return t.Time
}

func (t *FakeClock) Sleep(d time.Duration) {
	t.Lock()
	defer t.Unlock()
	t.Time = t.Time.Add(d)
}

type FakeAccessor struct {
	*core.DBCreator
	core.Fetcher
	Insights []int64
	ConnPair dbconn.ConnPair
}

func (c *FakeAccessor) GenerateInsight(tx *sql.Tx, properties core.InsightProperties) error {
	id, err := core.GenerateInsight(tx, properties)

	if err != nil {
		return errorutil.Wrap(err)
	}

	c.Insights = append(c.Insights, id)

	return nil
}

// NewFakeAccessor returns an acessor that implements core.Fetcher and core.Creator
// using a temporary database that should be delete using clear()
func NewFakeAccessor(t *testing.T) (acessor *FakeAccessor, clear func()) {
	connPair, removeDir := testutil.TempDBConnection(t, "insights")

	if err := migrator.Run(connPair.RwConn.DB, "insights"); err != nil {
		t.Fatal(err)
	}

	creator, err := core.NewCreator(connPair.RwConn)
	if err != nil {
		t.Fatal(err)
	}

	fetcher, err := core.NewFetcher(connPair.RoConn)
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
