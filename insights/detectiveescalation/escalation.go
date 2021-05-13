// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package detectiveescalation

import (
	"context"
	"database/sql"
	"errors"
	"gitlab.com/lightmeter/controlcenter/detective"
	"gitlab.com/lightmeter/controlcenter/detective/escalator"
	"gitlab.com/lightmeter/controlcenter/i18n/translator"
	"gitlab.com/lightmeter/controlcenter/insights/core"
	notificationCore "gitlab.com/lightmeter/controlcenter/notification/core"
	parser "gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"time"
)

type Options struct {
	Escalator escalator.Stepper
}

const (
	ContentType   = "detective_escalation"
	ContentTypeId = 8
)

func init() {
	core.RegisterContentType(ContentType, ContentTypeId, core.DefaultContentTypeDecoder(&Content{}))
}

type Content struct {
	Sender    string                `json:"sender"`
	Recipient string                `json:"recipient"`
	Interval  timeutil.TimeInterval `json:"time_interval"`
	Messages  detective.Messages    `json:"messages"`
}

func (c Content) Title() notificationCore.ContentComponent {
	return &title{c}
}

func (c Content) Description() notificationCore.ContentComponent {
	return &description{c}
}

func (c Content) Metadata() notificationCore.ContentMetadata {
	return nil
}

type title struct {
	c Content
}

func (t title) String() string {
	return translator.Stringfy(t)
}

func (t title) TplString() string {
	return translator.I18n("User request on non delivered message")
}

func (t title) Args() []interface{} {
	return nil
}

type description struct {
	c Content
}

func (d description) String() string {
	return translator.Stringfy(d)
}

func (d description) TplString() string {
	return translator.I18n("Sender: %v, recipient: %v")
}

func (d description) Args() []interface{} {
	return []interface{}{d.c.Sender, d.c.Recipient}
}

type detector struct {
	options Options
	creator core.Creator
}

func NewDetector(creator core.Creator, options core.Options) core.Detector {
	detectorOptions, ok := options["detective"].(Options)

	if !ok {
		errorutil.MustSucceed(errors.New("Invalid detector options"))
	}

	return &detector{
		options: detectorOptions,
		creator: creator,
	}
}

func maybeAddNewInsightFromMessage(d *detector, r escalator.Request, c core.Clock, tx *sql.Tx) error {
	content := Content{
		Sender:    r.Sender,
		Recipient: r.Recipient,
		Interval:  r.Interval,
		Messages:  r.Messages,
	}

	if err := generateInsight(tx, c, d.creator, content); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func (d *detector) Step(c core.Clock, tx *sql.Tx) error {
	return d.options.Escalator.Step(func(r escalator.Request) error {
		return maybeAddNewInsightFromMessage(d, r, c, tx)
	}, func() error {
		return nil
	})
}

func (d *detector) Close() error {
	return nil
}

// TODO: refactor this function to be reused across different insights instead of copy&pasted
func generateInsight(tx *sql.Tx, c core.Clock, creator core.Creator, content Content) error {
	if err := creator.GenerateInsight(context.Background(), tx, BuildInsightProperties(c, content)); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func BuildInsightProperties(c core.Clock, content Content) core.InsightProperties {
	return core.InsightProperties{
		Time:        c.Now(),
		Category:    core.LocalCategory,
		Rating:      core.Unrated,
		ContentType: ContentType,
		Content:     content,
	}
}

var SampleInsightContent = Content{
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
				Status:           detective.Status(parser.DeferredStatus),
				Dsn:              "3.0.0",
			},
			{
				NumberOfAttempts: 1,
				TimeMin:          timeutil.MustParseTime(`2000-01-01 10:00:00 +0000`),
				TimeMax:          timeutil.MustParseTime(`2000-01-01 10:00:00 +0000`),
				Status:           detective.Status(parser.ExpiredStatus),
				Dsn:              "4.0.0",
			},
		},
		"CCCCCCCCC": []detective.MessageDelivery{
			{
				NumberOfAttempts: 1,
				TimeMin:          timeutil.MustParseTime(`2000-01-03 10:00:00 +0000`),
				TimeMax:          timeutil.MustParseTime(`2000-01-03 10:00:00 +0000`),
				Status:           detective.Status(parser.BouncedStatus),
				Dsn:              "3.0.0",
			},
		},
	},
}
