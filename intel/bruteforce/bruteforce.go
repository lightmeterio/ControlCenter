// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package bruteforce

import (
	"time"
)

type Checker interface {
	Step(time.Time, func(SummaryResult) error) error
}

type BlockedIP struct {
	Addr  string `json:"addr"`
	Count int    `json:"count"`
}

type SummaryResult struct {
	TopIPs      []BlockedIP `json:"top_ips"`
	TotalNumber int         `json:"total_number"`
}
