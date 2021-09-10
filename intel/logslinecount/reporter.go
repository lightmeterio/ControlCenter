// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package logslinecount

import (
	"database/sql"
	"gitlab.com/lightmeter/controlcenter/intel/collector"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"time"
)

type payloadCounter struct {
	Supported   int `json:"supported"`
	Unsupported int `json:"unsupported"`
}

type report struct {
	Interval timeutil.TimeInterval     `json:"time_interval"`
	Counters map[string]payloadCounter `json:"counters"`
}

type Reporter struct {
	pub *Publisher
}

func NewReporter(pub *Publisher) *Reporter {
	return &Reporter{pub: pub}
}

const executionInterval = 10 * time.Minute

func (r *Reporter) ExecutionInterval() time.Duration {
	return executionInterval
}

func (r *Reporter) Close() error {
	return nil
}

func computeKey(k counterKey) string {
	if len(k.daemon) == 0 {
		return k.process
	}

	return k.process + "/" + k.daemon
}

func (r *Reporter) Step(tx *sql.Tx, clock timeutil.Clock) error {
	interval := timeutil.TimeInterval{From: clock.Now().Add(-executionInterval), To: clock.Now()}

	report := report{
		Interval: interval,
		Counters: map[string]payloadCounter{},
	}

	counters := map[counterKey]payloadCounter{}

	flushPublisher(r.pub, counters)

	total := 0

	for k, v := range counters {
		report.Counters[computeKey(k)] = v
		total += v.Supported + v.Unsupported
	}

	if total == 0 {
		return nil
	}

	if err := collector.Collect(tx, clock, r.ID(), &report); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func (r *Reporter) ID() string {
	return "log_lines_count"
}
