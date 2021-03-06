// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package mailinactivity

import (
	"context"
	"database/sql"
	"errors"
	"gitlab.com/lightmeter/controlcenter/i18n/translator"
	"gitlab.com/lightmeter/controlcenter/insights/core"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	notificationCore "gitlab.com/lightmeter/controlcenter/notification/core"
	insightsSettings "gitlab.com/lightmeter/controlcenter/settings/insights"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"time"
)

type Content struct {
	Interval timeutil.TimeInterval `json:"interval"`
}

func (c Content) Title() notificationCore.ContentComponent {
	return &title{}
}

func (c Content) Description() notificationCore.ContentComponent {
	return &description{c}
}

func (c Content) Metadata() notificationCore.ContentMetadata {
	return nil
}

type title struct{}

func (t title) String() string {
	return translator.Stringfy(t)
}

func (title) TplString() string {
	return translator.I18n("Mail Inactivity")
}

func (title) Args() []interface{} {
	return nil
}

type description struct {
	c Content
}

func (d description) String() string {
	return translator.Stringfy(d)
}

func (d description) TplString() string {
	return translator.I18n("No emails were sent or received between %v and %v")
}

func (d description) Args() []interface{} {
	return []interface{}{d.c.Interval.From, d.c.Interval.To}
}

func (c Content) HelpLink(urlContainer core.URLContainer) string {
	return urlContainer.Get(ContentType)
}

type generator struct {
	creator  core.Creator
	interval *timeutil.TimeInterval
}

func (*generator) Close() error {
	return nil
}

const (
	ContentType   = "mail_inactivity"
	ContentTypeId = 0
)

type Options struct {
	LookupRange               time.Duration
	MinTimeGenerationInterval time.Duration
}

// TODO: get the value inserted by the detector from the db, if there's any,
// and use it to generate a new insight

func (g *generator) Step(c core.Clock, tx *sql.Tx) error {
	if g.interval == nil {
		return nil
	}

	if err := generateInsight(tx, c, g.creator, *g.interval); err != nil {
		return errorutil.Wrap(err)
	}

	g.interval = nil

	return nil
}

func (g *generator) generate(interval timeutil.TimeInterval) {
	g.interval = &interval
}

type Detector struct {
	logsConnPool *dbconn.RoPool
	options      Options
	creator      core.Creator
	generator    *generator
}

func (Detector) IsHistoricalDetector() {
	// Required by the historical import
}

func (*Detector) Close() error {
	return nil
}

func (d *Detector) GetOptions() Options {
	return d.options
}

const countDeliveriesInIntervalQueryKey = "countDeliveriesInInterval"

func (d *Detector) UpdateOptionsFromSettings(settings *insightsSettings.Settings) {
	d.options.LookupRange = time.Hour * time.Duration(settings.MailInactivityLookupRange)
	d.options.MinTimeGenerationInterval = time.Hour * time.Duration(settings.MailInactivityMinInterval)
}

func NewDetector(settings *insightsSettings.Settings, creator core.Creator, options core.Options) core.Detector {
	pool, ok := options["logsConnPool"].(*dbconn.RoPool)

	if !ok {
		errorutil.MustSucceed(errors.New("Invalid Connection Pool"))
	}

	errorutil.MustSucceed(pool.ForEach(func(conn *dbconn.RoPooledConn) error {
		if err := conn.PrepareStmt(`select count(*) from deliveries where delivery_ts between ? and ?`, countDeliveriesInIntervalQueryKey); err != nil {
			return errorutil.Wrap(err)
		}

		return nil
	}))

	detector := &Detector{
		logsConnPool: pool,
		creator:      creator,
		generator:    &generator{creator: creator, interval: nil},
	}

	detector.UpdateOptionsFromSettings(settings)

	return detector
}

func execChecksForMailInactivity(ctx context.Context, d *Detector, c core.Clock, tx *sql.Tx) error {
	now := c.Now()

	kind := "mail_inactivity"

	lastExecTime, err := core.RetrieveLastDetectorExecution(tx, kind)

	if err != nil {
		return errorutil.Wrap(err)
	}

	interval := timeutil.TimeInterval{
		From: now.Add(-d.options.LookupRange),
		To:   now,
	}

	// no time: first execution, does the check
	// time less than one minute: does nothing
	// time greater than one minute: execute generator

	if !(lastExecTime.IsZero() || (!lastExecTime.IsZero() && now.Sub(lastExecTime) >= d.options.MinTimeGenerationInterval)) {
		return nil
	}

	conn, release, err := d.logsConnPool.AcquireContext(ctx)
	if err != nil {
		return errorutil.Wrap(err)
	}

	defer release()

	countActivityInInterval := func(interval timeutil.TimeInterval) (int, error) {
		var count int
		//nolint:sqlclosecheck
		if err := conn.GetStmt(countDeliveriesInIntervalQueryKey).QueryRowContext(ctx, interval.From.Unix(), interval.To.Unix()).Scan(&count); err != nil {
			return 0, errorutil.Wrap(err)
		}

		return count, nil
	}

	totalCurrentInterval, err := countActivityInInterval(interval)
	if err != nil {
		return errorutil.Wrap(err)
	}

	if totalCurrentInterval > 0 {
		return nil
	}

	if lastExecTime.IsZero() {
		// potentially first insight generation
		totalPreviousInterval, err := countActivityInInterval(timeutil.TimeInterval{
			From: interval.From.Add(d.options.LookupRange * -1),
			To:   interval.To.Add(d.options.LookupRange * -1),
		})

		if err != nil {
			return errorutil.Wrap(err)
		}

		if totalCurrentInterval == 0 && totalPreviousInterval == 0 {
			return nil
		}
	}

	d.generator.generate(interval)

	if err := core.StoreLastDetectorExecution(tx, kind, now); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func (d *Detector) Step(c core.Clock, tx *sql.Tx) error {
	ctx := context.Background()

	if err := execChecksForMailInactivity(ctx, d, c, tx); err != nil {
		return errorutil.Wrap(err)
	}

	if err := d.generator.Step(c, tx); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func generateInsight(tx *sql.Tx, c core.Clock, creator core.Creator, interval timeutil.TimeInterval) error {
	properties := core.InsightProperties{
		Time:        c.Now(),
		Category:    core.LocalCategory,
		Rating:      core.OkRating,
		ContentType: ContentType,
		Content: Content{
			Interval: interval,
		},
		MustBeNotified: true,
	}

	if err := creator.GenerateInsight(context.Background(), tx, properties); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func init() {
	core.RegisterContentType(ContentType, ContentTypeId, core.DefaultContentTypeDecoder(&Content{}))
}
