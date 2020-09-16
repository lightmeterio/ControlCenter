package testutil

import (
	"database/sql"
	"gitlab.com/lightmeter/controlcenter/insights/core"
	"time"
)

type FakeClock struct {
	time.Time
}

func (t *FakeClock) Now() time.Time {
	return t.Time
}

func (t *FakeClock) Sleep(d time.Duration) {
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
