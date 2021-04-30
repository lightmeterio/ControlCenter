// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

// +build dev !release

package detectiveescalation

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
		Sender:    "sender@example.com",
		Recipient: "recipient@example.com",
		Interval: timeutil.TimeInterval{
			From: time.Date(time.Now().Year(), time.January, 1, 0, 0, 0, 0, time.UTC),
			To:   time.Date(time.Now().Year(), time.December, 31, 23, 59, 59, 59, time.UTC),
		},
	}); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}
