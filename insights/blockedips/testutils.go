// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package blockedips

import (
	"gitlab.com/lightmeter/controlcenter/intel/blockedips"
	"time"
)

type FakeChecker struct {
	Actions map[time.Time]blockedips.SummaryResult
}

func (c *FakeChecker) Step(now time.Time, withResults func(blockedips.SummaryResult) error) error {
	if result, ok := c.Actions[now]; ok {
		return withResults(result)
	}

	return nil
}
