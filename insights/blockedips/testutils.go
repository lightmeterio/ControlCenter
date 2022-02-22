// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package blockedips

import (
	"gitlab.com/lightmeter/controlcenter/intel/blockedips"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
)

type FakeChecker struct {
	Actions map[timeutil.TimeInterval]blockedips.SummaryResult
}

func (c *FakeChecker) Step(interval timeutil.TimeInterval, withResults func(blockedips.SummaryResult) error) error {
	if result, ok := c.Actions[interval]; ok {
		return withResults(result)
	}

	return nil
}
