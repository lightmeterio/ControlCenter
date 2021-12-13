// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package blockedipssummary

import (
	"context"
	"database/sql"
	"errors"
	"github.com/rs/zerolog/log"
	"gitlab.com/lightmeter/controlcenter/i18n/translator"
	"gitlab.com/lightmeter/controlcenter/insights/blockedips"
	"gitlab.com/lightmeter/controlcenter/insights/core"
	notificationCore "gitlab.com/lightmeter/controlcenter/notification/core"
	"gitlab.com/lightmeter/controlcenter/util/closeutil"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"math"
	"time"
)

const (
	ContentType   = "blockedips_summary"
	ContentTypeId = 10
)

type Options struct {
	TimeSpan        time.Duration
	InsightsFetcher core.Fetcher
}

type Summary struct {
	Interval         timeutil.TimeInterval `json:"time_interval"`
	IPCount          int                   `json:"ip_count"`
	ConnectionsCount int                   `json:"connections_count"`
	RefID            int                   `json:"ref_id"`
}

type Content struct {
	Interval timeutil.TimeInterval `json:"time_interval"`
	Summary  []Summary             `json:"summary"`
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
	return translator.I18n("Suspicious IPs banned last week")
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
	return translator.I18n("%v Connections from %v IPs were blocked over %v days")
}

func (d description) Args() []interface{} {
	days := math.Round(d.c.Interval.To.Sub(d.c.Interval.From).Round(24*time.Hour).Hours() / 24.)

	var (
		connCount int
		ipCount   int
	)

	for _, i := range d.c.Summary {
		connCount += i.ConnectionsCount
		ipCount += i.IPCount
	}

	return []interface{}{connCount, ipCount, days}
}

func (c Content) HelpLink(urlContainer core.URLContainer) string {
	return urlContainer.Get(ContentType)
}

func init() {
	core.RegisterContentType(ContentType, ContentTypeId, core.DefaultContentTypeDecoder(&Content{}))
}

type detector struct {
	closeutil.Closers

	options Options
	creator core.Creator
}

func getDetectorOptions(options core.Options) Options {
	detectorOptions, ok := options["blockedips_summary"].(Options)

	if !ok {
		errorutil.MustSucceed(errors.New("Invalid detector options"))
	}

	return detectorOptions
}

func NewDetector(creator core.Creator, options core.Options) core.Detector {
	detectorOptions := getDetectorOptions(options)

	if detectorOptions.InsightsFetcher == nil {
		errorutil.MustSucceed(errors.New("Invalid Fetcher"))
	}

	return &detector{
		Closers: closeutil.New(),
		options: detectorOptions,
		creator: creator,
	}
}

func (d *detector) Step(c core.Clock, tx *sql.Tx) error {
	now := c.Now()

	kind := "blockedips_summary"

	lastExecTime, err := core.RetrieveLastDetectorExecution(tx, kind)

	if err != nil {
		return errorutil.Wrap(err)
	}

	// first execution. Do not execute detector, but add a "mark" for when it's started
	if lastExecTime.IsZero() {
		if err := core.StoreLastDetectorExecution(tx, kind, now); err != nil {
			return errorutil.Wrap(err)
		}

		return nil
	}

	interval := timeutil.TimeInterval{
		From: now.Add(-d.options.TimeSpan),
		To:   now,
	}

	if now.Sub(lastExecTime) < d.options.TimeSpan {
		// too early to execute it again
		return nil
	}

	insights, err := d.options.InsightsFetcher.FetchInsights(context.Background(), core.FetchOptions{
		Interval: interval,
		OrderBy:  core.OrderByCreationAsc,
	}, c)

	if err != nil {
		return errorutil.Wrap(err)
	}

	//nolint:prealloc
	var summaries []Summary = nil

	for _, i := range insights {
		if i.ContentType() != blockedips.ContentType {
			continue
		}

		c, ok := i.Content().(*blockedips.Content)
		if !ok {
			log.Panic().Msg("Unable to get content of blockedips insight. This is a programming error!")
		}

		summaries = append(summaries, Summary{
			Interval:         c.Interval,
			IPCount:          c.TotalIPs,
			ConnectionsCount: c.TotalNumber,
			RefID:            i.ID(),
		})
	}

	if len(summaries) == 0 {
		if err := core.StoreLastDetectorExecution(tx, kind, now); err != nil {
			return errorutil.Wrap(err)
		}

		return nil
	}

	if err := generateInsight(tx, c, d.creator, Content{
		Interval: interval,
		Summary:  summaries,
	}); err != nil {
		return errorutil.Wrap(err)
	}

	if err := core.StoreLastDetectorExecution(tx, kind, now); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func generateInsight(tx *sql.Tx, c core.Clock, creator core.Creator, content Content) error {
	properties := core.InsightProperties{
		Time:           c.Now(),
		Category:       core.IntelCategory,
		Rating:         core.OkRating,
		ContentType:    ContentType,
		Content:        content,
		MustBeNotified: true,
	}

	if err := creator.GenerateInsight(context.Background(), tx, properties); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}
