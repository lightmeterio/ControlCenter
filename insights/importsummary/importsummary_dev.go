// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

//go:build dev || !release
// +build dev !release

package importsummary

import (
	"context"
	"database/sql"
	"gitlab.com/lightmeter/controlcenter/insights/core"
	"gitlab.com/lightmeter/controlcenter/insights/highrate"
	"gitlab.com/lightmeter/controlcenter/insights/localrbl"
	"gitlab.com/lightmeter/controlcenter/insights/mailinactivity"
	"gitlab.com/lightmeter/controlcenter/insights/messagerbl"
	"gitlab.com/lightmeter/controlcenter/localrbl"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"net"
)

// Executed only on development builds, for better developer experience
func (d *detector) GenerateSampleInsight(tx *sql.Tx, c core.Clock) error {
	insights := []ImportedInsight{
		{
			ID: 1, Time: testutil.MustParseTime(`2000-01-01 00:00:00 +0000`),
			Category: core.LocalCategory, Rating: core.BadRating,
			Content: messagerblinsight.Content{
				Address:   net.ParseIP(`8.8.8.8`),
				Recipient: "recipient.com",
				Host:      "Some Big Host",
				Status:    "bounced",
				Message:   "This is a fake message, really",
				Time:      testutil.MustParseTime(`2000-01-01 00:00:00 +0000`),
			},
			ContentType: messagerblinsight.ContentType,
		},
		{
			ID: 2, Time: testutil.MustParseTime(`2000-01-01 00:00:01 +0000`),
			Category: core.LocalCategory, Rating: core.BadRating,
			Content: localrblinsight.Content{
				ScanInterval: timeutil.TimeInterval{
					From: testutil.MustParseTime(`2000-01-01 00:00:00 +0000`),
					To:   testutil.MustParseTime(`2000-01-02 00:00:00 +0000`),
				},
				Address: net.ParseIP(`1.2.3.4`),
				RBLs: []localrbl.ContentElement{
					{RBL: "rbl1.com", Text: "Blocked due something bad"},
					{RBL: "lala.com", Text: "Some sample message for too much spam!!!"},
				},
			},
			ContentType: localrblinsight.ContentType,
		},
		{
			ID: 3, Time: testutil.MustParseTime(`2000-01-01 00:00:01 +0000`),
			Category: core.LocalCategory, Rating: core.BadRating,
			Content: highrate.BounceRateContent{
				Value: 0.5,
				Interval: timeutil.TimeInterval{
					From: testutil.MustParseTime(`2000-01-01 00:00:00 +0000`),
					To:   testutil.MustParseTime(`2000-01-02 00:00:00 +0000`),
				},
			},
			ContentType: highrate.HighBaseBounceRateContentType,
		},
		{
			ID: 4, Time: testutil.MustParseTime(`2000-01-01 00:00:01 +0000`),
			Category: core.LocalCategory, Rating: core.BadRating,
			Content: mailinactivity.Content{
				Interval: timeutil.TimeInterval{
					From: testutil.MustParseTime(`2000-01-01 00:00:00 +0000`),
					To:   testutil.MustParseTime(`2000-01-02 00:00:00 +0000`),
				},
			},
			ContentType: mailinactivity.ContentType,
		},
	}

	properties := core.InsightProperties{
		Time:        c.Now(),
		Category:    core.LocalCategory,
		Rating:      core.Unrated,
		ContentType: ContentType,
		Content: Content{
			Interval: timeutil.TimeInterval{
				From: testutil.MustParseTime(`2000-01-01 00:00:00 +0000`),
				To:   testutil.MustParseTime(`2000-01-02 00:00:00 +0000`),
			},
			Insights: insights,
		},
	}

	if _, err := core.GenerateInsight(context.Background(), tx, properties); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}
