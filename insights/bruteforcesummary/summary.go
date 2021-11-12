// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package bruteforcesummary

import (
	"context"
	"database/sql"
	"errors"
	"gitlab.com/lightmeter/controlcenter/i18n/translator"
	"gitlab.com/lightmeter/controlcenter/insights/core"
	"gitlab.com/lightmeter/controlcenter/intel/bruteforce"
	notificationCore "gitlab.com/lightmeter/controlcenter/notification/core"
	"gitlab.com/lightmeter/controlcenter/util/closeutil"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"time"
)

type Options struct {
	Checker      bruteforce.Checker
	PollInterval time.Duration
}

type Content bruteforce.SummaryResult

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
	return translator.I18n("Attacks were prevented")
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
	return translator.I18n("Network attacks were blocked: %d")
}

func (d description) Args() []interface{} {
	return []interface{}{d.c.TotalNumber}
}

func (c Content) HelpLink(urlContainer core.URLContainer) string {
	return urlContainer.Get(ContentType)
}

const (
	ContentType   = "bruteforcesummary"
	ContentTypeId = 9
)

func init() {
	core.RegisterContentType(ContentType, ContentTypeId, core.DefaultContentTypeDecoder(&Content{}))
}

type detector struct {
	closeutil.Closers

	options Options
	creator core.Creator
}

func getDetectorOptions(options core.Options) Options {
	detectorOptions, ok := options["bruteforcesummary"].(Options)

	if !ok {
		errorutil.MustSucceed(errors.New("Invalid detector options"))
	}

	return detectorOptions
}

func NewDetector(creator core.Creator, options core.Options) core.Detector {
	detectorOptions := getDetectorOptions(options)

	return &detector{
		Closers: closeutil.New(),
		options: detectorOptions,
		creator: creator,
	}
}

func archiveAnyPreviousInsightIfNeeded(tx *sql.Tx, c core.Clock) error {
	var id int64

	err := tx.QueryRow(`select rowid from insights where content_type = ? order by rowid desc limit 1`, ContentTypeId).Scan(&id)

	if err != nil && errors.Is(err, sql.ErrNoRows) {
		return nil
	}

	if err != nil {
		return errorutil.Wrap(err)
	}

	if err := core.ArchiveInsight(context.Background(), tx, id, c.Now()); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func (d *detector) Step(c core.Clock, tx *sql.Tx) error {
	return d.options.Checker.Step(c.Now(), func(r bruteforce.SummaryResult) error {
		if err := archiveAnyPreviousInsightIfNeeded(tx, c); err != nil {
			return errorutil.Wrap(err)
		}

		return generateInsight(tx, c, d.creator, Content{TopIPs: r.TopIPs, TotalNumber: r.TotalNumber})
	})
}

func generateInsight(tx *sql.Tx, c core.Clock, creator core.Creator, content Content) error {
	properties := core.InsightProperties{
		Time:        c.Now(),
		Category:    core.IntelCategory,
		Rating:      core.OkRating,
		ContentType: ContentType,
		Content:     content,
	}

	if err := creator.GenerateInsight(context.Background(), tx, properties); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}
