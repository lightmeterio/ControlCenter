// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

// +build dev !release

package detectiveescalation

import (
	"database/sql"
	"gitlab.com/lightmeter/controlcenter/detective"
	"gitlab.com/lightmeter/controlcenter/insights/core"
	parser "gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser"
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
		Messages: detective.Messages{
			"AAAAAAAAA": []detective.MessageDelivery{
				{
					NumberOfAttempts: 30,
					TimeMin:          timeutil.MustParseTime(`2000-01-01 00:00:00 +0000`),
					TimeMax:          timeutil.MustParseTime(`2000-01-01 10:00:00 +0000`),
					Status:           parser.DeferredStatus,
					Dsn:              "3.0.0",
				},
				{
					NumberOfAttempts: 1,
					TimeMin:          timeutil.MustParseTime(`2000-01-01 10:00:00 +0000`),
					TimeMax:          timeutil.MustParseTime(`2000-01-01 10:00:00 +0000`),
					Status:           parser.ExpiredStatus,
					Dsn:              "4.0.0",
				},
			},
			"BBBBBBBBBB": []detective.MessageDelivery{
				{
					NumberOfAttempts: 20,
					TimeMin:          timeutil.MustParseTime(`2000-01-01 00:00:00 +0000`),
					TimeMax:          timeutil.MustParseTime(`2000-01-01 10:00:00 +0000`),
					Status:           parser.DeferredStatus,
					Dsn:              "4.0.0",
				},
				{
					NumberOfAttempts: 1,
					TimeMin:          timeutil.MustParseTime(`2000-01-02 10:00:00 +0000`),
					TimeMax:          timeutil.MustParseTime(`2000-01-02 10:00:00 +0000`),
					Status:           parser.SentStatus,
					Dsn:              "2.0.0",
				},
			},
			"CCCCCCCCC": []detective.MessageDelivery{
				{
					NumberOfAttempts: 1,
					TimeMin:          timeutil.MustParseTime(`2000-01-03 10:00:00 +0000`),
					TimeMax:          timeutil.MustParseTime(`2000-01-03 10:00:00 +0000`),
					Status:           parser.BouncedStatus,
					Dsn:              "3.0.0",
				},
			},
		},
	}); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}
