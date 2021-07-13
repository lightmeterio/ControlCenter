// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package insights

import (
	"context"
	"database/sql"
	"gitlab.com/lightmeter/controlcenter/insights/core"
	"gitlab.com/lightmeter/controlcenter/intel/collector"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"time"
)

type report struct {
	Interval timeutil.TimeInterval `json:"time_interval"`
	Insights map[string]int        `json:"insights"`
}

type Reporter struct {
	fetcher core.Fetcher
}

func NewReporter(fetcher core.Fetcher) *Reporter {
	return &Reporter{fetcher: fetcher}
}

const executionInterval = 15 * time.Minute

func (r *Reporter) ExecutionInterval() time.Duration {
	return executionInterval
}

func (r *Reporter) Close() error {
	return nil
}

func (r *Reporter) Step(tx *sql.Tx, clock timeutil.Clock) error {
	interval := timeutil.TimeInterval{From: clock.Now().Add(-executionInterval), To: clock.Now()}

	report := report{
		Interval: interval,
		Insights: map[string]int{},
	}

	fetchedInsights, err := r.fetcher.FetchInsights(context.Background(), core.FetchOptions{
		Interval: interval,
	}, clock)

	if err != nil {
		return errorutil.Wrap(err)
	}

	for _, insight := range fetchedInsights {
		insightType := insight.ContentType()
		if _, ok := report.Insights[insightType]; !ok {
			report.Insights[insightType] = 0
		}
		report.Insights[insightType]++
	}

	if err := collector.Collect(tx, clock, r.ID(), &report); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func (r *Reporter) ID() string {
	return "insights_count"
}
