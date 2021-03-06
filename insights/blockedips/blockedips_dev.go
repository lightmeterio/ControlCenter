// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

//go:build dev || !release
// +build dev !release

package blockedips

import (
	"database/sql"
	"gitlab.com/lightmeter/controlcenter/insights/core"
	"gitlab.com/lightmeter/controlcenter/intel/blockedips"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"time"
)

// Executed only on development builds, for better developer experience
func (d *detector) GenerateSampleInsight(tx *sql.Tx, c core.Clock) error {
	if err := generateInsight(tx, c, d.creator, Content{
		Interval: timeutil.TimeInterval{From: c.Now().Add(-24 * time.Hour), To: c.Now()},
		TopIPs: []blockedips.BlockedIP{
			{Address: "55.44.33.22", Count: 100},
			{Address: "11.22.33.44", Count: 45},
			{Address: "5.6.7.8", Count: 35},
		},
		TotalNumber: 234,
		TotalIPs:    10,
	}); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}
