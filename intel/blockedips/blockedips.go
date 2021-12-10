// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package blockedips

import (
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"time"
)

type Checker interface {
	Step(time.Time, func(SummaryResult) error) error
}

type BlockedIP struct {
	Address string `json:"addr"`
	Count   int    `json:"count"`
}

type SummaryResult struct {
	Interval    timeutil.TimeInterval `json:"time_interval"`
	TopIPs      []BlockedIP           `json:"top_ips"`
	TotalNumber int                   `json:"total_number"`
	TotalIPs    int                   `json:"total_ips"`
}
