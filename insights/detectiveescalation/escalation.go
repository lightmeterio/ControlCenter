// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package detectiveescalation

import (
	"context"
	"database/sql"
	"errors"
	"gitlab.com/lightmeter/controlcenter/detective/escalator"
	"gitlab.com/lightmeter/controlcenter/i18n/translator"
	"gitlab.com/lightmeter/controlcenter/insights/core"
	notificationCore "gitlab.com/lightmeter/controlcenter/notification/core"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
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
	return translator.I18n("Detective escalation request")
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
	return translator.I18n("Escalation request with sender: %v, recipient: %v, from %v to %v")
}

func (d description) Args() []interface{} {
	return []interface{}{d.c.Sender, d.c.Recipient, d.c.Interval.From, d.c.Interval.To}
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
	properties := core.InsightProperties{
		Time:        c.Now(),
		Category:    core.LocalCategory,
		Rating:      core.Unrated,
		ContentType: ContentType,
		Content:     content,
	}

	if err := creator.GenerateInsight(context.Background(), tx, properties); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}
