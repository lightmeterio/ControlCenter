// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

//go:build dev || !release
// +build dev !release

package blockedipssummary

import (
	"database/sql"
	"gitlab.com/lightmeter/controlcenter/insights/core"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"time"
)

// Executed only on development builds, for better developer experience
func (d *detector) GenerateSampleInsight(tx *sql.Tx, c core.Clock) error {
	if err := generateInsight(tx, c, d.creator, Content{
		Interval: timeutil.TimeInterval{From: c.Now().Add(-24 * time.Hour * 7), To: c.Now()},
		Summary: []Summary{
			{
				Interval:         timeutil.TimeInterval{From: c.Now().Add(-24 * time.Hour * 3), To: c.Now().Add(-24 * time.Hour * 2)},
				IPCount:          42,
				ConnectionsCount: 1056,
				RefID:            4,
			},
			{
				Interval:         timeutil.TimeInterval{From: c.Now().Add(-24 * time.Hour * 4), To: c.Now().Add(-24 * time.Hour * 3)},
				IPCount:          35,
				ConnectionsCount: 20567,
				RefID:            6,
			},
			{
				Interval:         timeutil.TimeInterval{From: c.Now().Add(-24 * time.Hour * 5), To: c.Now().Add(-24 * time.Hour * 5)},
				IPCount:          17,
				ConnectionsCount: 4035,
				RefID:            7,
			},
		},
	}); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}
