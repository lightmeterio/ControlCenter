// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package importsummary

import (
	"context"
	"database/sql"
	"github.com/rs/zerolog/log"
	"gitlab.com/lightmeter/controlcenter/i18n/translator"
	"gitlab.com/lightmeter/controlcenter/insights/core"
	notificationCore "gitlab.com/lightmeter/controlcenter/notification/core"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
)

type detector struct {
	fetcher  core.Fetcher
	interval timeutil.TimeInterval
}

func (detector) Close() error {
	return nil
}

func NewDetector(fetcher core.Fetcher, interval timeutil.TimeInterval) core.Detector {
	return &detector{fetcher: fetcher, interval: interval}
}

type title struct {
	c Content
}

func (t title) String() string {
	return translator.Stringfy(t)
}

func (t title) TplString() string {
	return translator.I18n("Imported insights")
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
	return translator.I18n("From %s to %d %d insights were imported")
}

func (d description) Args() []interface{} {
	return []interface{}{d.c.Interval.From, d.c.Interval.To, len(d.c.IDs)}
}

type Content struct {
	Interval timeutil.TimeInterval `json:"interval"`
	IDs      []int                 `json:"ids"`
}

func (c Content) Title() notificationCore.ContentComponent {
	return title{c}
}

func (c Content) Description() notificationCore.ContentComponent {
	return description{c}
}

func (c Content) Metadata() notificationCore.ContentMetadata {
	return nil
}

func (d *detector) Step(c core.Clock, tx *sql.Tx) error {
	var count int

	if err := tx.QueryRow(`select count(content) from insights where content_type = ?`, ContentTypeId).Scan(&count); err != nil {
		return errorutil.Wrap(err)
	}

	if count > 0 {
		log.Info().Msgf("Historical insights were already generated. Skipping")
		return nil
	}

	insights, err := d.fetcher.FetchInsights(context.Background(), core.FetchOptions{
		Interval: d.interval,
		OrderBy:  core.OrderByCreationAsc,
	})

	if err != nil {
		return errorutil.Wrap(err)
	}

	ids := []int{}

	for _, i := range insights {
		ids = append(ids, i.ID())
	}

	properties := core.InsightProperties{
		Time:        c.Now(),
		Category:    core.LocalCategory,
		Rating:      core.Unrated,
		ContentType: ContentType,
		Content: Content{
			Interval: d.interval,
			IDs:      ids,
		},
	}

	if _, err := core.GenerateInsight(context.Background(), tx, properties); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

const (
	ContentType   = "import_summary"
	ContentTypeId = 7
)

func init() {
	core.RegisterContentType(ContentType, ContentTypeId, core.DefaultContentTypeDecoder(&Content{}))
}
