package testutil

import (
	"database/sql"
	"gitlab.com/lightmeter/controlcenter/insights/core"
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
}

func (c *FakeAcessor) GenerateInsight(tx *sql.Tx, properties core.InsightProperties) error {
	id, err := core.GenerateInsight(tx, properties)

	if err != nil {
		return err
	}

	c.Insights = append(c.Insights, id)

	return nil
}
