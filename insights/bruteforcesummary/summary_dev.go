// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

//go:build dev || !release
// +build dev !release

package bruteforcesummary

import (
	"database/sql"
	"gitlab.com/lightmeter/controlcenter/insights/core"
	"gitlab.com/lightmeter/controlcenter/intel/bruteforce"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"time"
)

// Executed only on development builds, for better developer experience
func (d *detector) GenerateSampleInsight(tx *sql.Tx, c core.Clock) error {
	if err := generateInsight(tx, c, d.creator, Content{
		Interval: timeutil.TimeInterval{From: c.Now().Add(-24 * time.Hour), To: c.Now()},
		TopIPs: []bruteforce.BlockedIP{
			{Address: "55.44.33.22", Count: 100},
			{Address: "11.22.33.44", Count: 45},
		},
		TotalNumber: 234,
	}); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}
