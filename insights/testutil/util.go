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
	"log"
	"sync"
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

type FakeAcessor struct {
	*core.DBCreator
	core.Fetcher
	Insights []int64
	ConnPair dbconn.ConnPair
}

func (c *FakeAcessor) GenerateInsight(tx *sql.Tx, properties core.InsightProperties) error {
	id, err := core.GenerateInsight(tx, properties)

	if err != nil {
		return errorutil.Wrap(err)
	}

	c.Insights = append(c.Insights, id)

	return nil
}

// NewFakeAcessor returns an acessor that implements core.Fetcher and core.Creator
// using a temporary database that should be delete using clear()
func NewFakeAcessor() (acessor *FakeAcessor, clear func()) {
	connPair, removeDir := testutil.TempDBConnection("insights")

	if err := migrator.Run(connPair.RwConn.DB, "insights"); err != nil {
		log.Panicln(err)
	}

	creator, err := core.NewCreator(connPair.RwConn)
	if err != nil {
		log.Panicln(err)
	}

	fetcher, err := core.NewFetcher(connPair.RoConn)
	if err != nil {
		log.Panicln(err)
	}

	return &FakeAcessor{DBCreator: creator, Fetcher: fetcher, Insights: []int64{}, ConnPair: connPair},
		func() {
			if connPair.Close(); err != nil {
				log.Panicln(err)
			}

			removeDir()
		}
}
