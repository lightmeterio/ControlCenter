// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package blockedips

import (
	"context"
	"database/sql"
	"errors"
	"gitlab.com/lightmeter/controlcenter/i18n/translator"
	"gitlab.com/lightmeter/controlcenter/insights/core"
	"gitlab.com/lightmeter/controlcenter/intel/blockedips"
	notificationCore "gitlab.com/lightmeter/controlcenter/notification/core"
	"gitlab.com/lightmeter/controlcenter/pkg/closers"
	insightsSettings "gitlab.com/lightmeter/controlcenter/settings/insights"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"time"
)

const (
	ContentType   = "blockedips"
	ContentTypeId = 9
)

type Options struct {
	Checker      blockedips.Checker
	PollInterval time.Duration
}

type Content blockedips.SummaryResult

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
	return translator.I18n("Blocked suspicious connection attempts")
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
	return translator.I18n("%v connections blocked from %v banned IPs (peer network)")
}

func (d description) Args() []interface{} {
	return []interface{}{d.c.TotalNumber, d.c.TotalIPs}
}

func (c Content) HelpLink(urlContainer core.URLContainer) string {
	return urlContainer.Get(ContentType)
}

func init() {
	core.RegisterContentType(ContentType, ContentTypeId, core.DefaultContentTypeDecoder(&Content{}))
}

type detector struct {
	closers.Closers

	options Options
	creator core.Creator
}

func getDetectorOptions(options core.Options) Options {
	detectorOptions, ok := options["blockedips"].(Options)

	if !ok {
		errorutil.MustSucceed(errors.New("Invalid detector options"))
	}

	return detectorOptions
}

func (d *detector) UpdateOptionsFromSettings(settings *insightsSettings.Settings) {}

func NewDetector(creator core.Creator, options core.Options) core.Detector {
	detectorOptions := getDetectorOptions(options)

	return &detector{
		Closers: closers.New(),
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
	now := c.Now()

	kind := "blockedips"

	lastExecTime, err := core.RetrieveLastDetectorExecution(tx, kind)
	if err != nil {
		return errorutil.Wrap(err)
	}

	// respect the polling time
	if !(lastExecTime.IsZero() || now.Sub(lastExecTime) >= d.options.PollInterval) {
		return nil
	}

	if err := core.StoreLastDetectorExecution(tx, kind, now); err != nil {
		return errorutil.Wrap(err)
	}

	return d.options.Checker.Step(c.Now(), func(r blockedips.SummaryResult) error {
		if err := archiveAnyPreviousInsightIfNeeded(tx, c); err != nil {
			return errorutil.Wrap(err)
		}

		return generateInsight(tx, c, d.creator, Content(r))
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
