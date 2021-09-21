// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

//go:build dev || !release
// +build dev !release

package highrate

import (
	"database/sql"
	"gitlab.com/lightmeter/controlcenter/insights/core"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
)

// Executed only on development builds, for better developer experience
func (d *highRateDetector) GenerateSampleInsight(tx *sql.Tx, c core.Clock) error {
	for _, g := range d.generators {
		now := c.Now()

		content := BounceRateContent{
			Value:    0.9,
			Interval: timeutil.TimeInterval{From: now.Add(g.checkTimespan * -1), To: now},
		}

		if err := generateInsight(tx, c, g.creator, content); err != nil {
			return errorutil.Wrap(err)
		}
	}

	return nil
}
