// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package highrate

import (
	"context"
	"database/sql"
	"errors"
	"gitlab.com/lightmeter/controlcenter/dashboard"
	"gitlab.com/lightmeter/controlcenter/i18n/translator"
	"gitlab.com/lightmeter/controlcenter/insights/core"
	notificationCore "gitlab.com/lightmeter/controlcenter/notification/core"
	insightsSettings "gitlab.com/lightmeter/controlcenter/settings/insights"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"time"
)

const (
	HighBaseBounceRateContentType   = "high_bounce_rate"
	HighBaseBounceRateContentTypeId = 1
)

type Options struct {
	BaseBounceRateThreshold int
}

type bounceRateGenerator struct {
	creator                     core.Creator
	value                       *BounceRateContent
	minimalNotiticationInterval time.Duration
	checkTimespan               time.Duration
	kind                        string
}

func (*bounceRateGenerator) Close() error {
	return nil
}

func (g *bounceRateGenerator) Step(c core.Clock, tx *sql.Tx) error {
	if g.value == nil {
		return nil
	}

	if err := generateInsight(tx, c, g.creator, *g.value); err != nil {
		return errorutil.Wrap(err)
	}

	g.value = nil

	return nil
}

func (g *bounceRateGenerator) generate(interval timeutil.TimeInterval, value float32) {
	g.value = &BounceRateContent{Value: value, Interval: interval}
}

type Detector struct {
	bounceRateThreshold float32
	dashboard           dashboard.Dashboard
	generators          []*bounceRateGenerator
}

func (*Detector) IsHistoricalDetector() {
	// Required by the historical import
}

func (*Detector) Close() error {
	return nil
}

func (d *Detector) UpdateOptionsFromSettings(settings *insightsSettings.Settings) {
	d.bounceRateThreshold = float32(settings.BounceRateThreshold) / 100
}

func (d *Detector) GetBounceRateThreshold() float32 {
	return d.bounceRateThreshold
}

func NewDetector(settings *insightsSettings.Settings, creator core.Creator, options core.Options) core.Detector {
	dashboard, ok := options["dashboard"].(dashboard.Dashboard)
	if !ok {
		errorutil.MustSucceed(errors.New("Invalid dashboard"))
	}

	detector := &Detector{
		dashboard: dashboard,
		generators: []*bounceRateGenerator{
			{
				creator:                     creator,
				checkTimespan:               time.Hour * 6,
				minimalNotiticationInterval: time.Hour * 2,
				value:                       nil,
				kind:                        "high_base_bounce_rate",
			},
		},
	}

	detector.UpdateOptionsFromSettings(settings)

	return detector
}

func tryToDetectAndGenerateInsight(ctx context.Context, gen *bounceRateGenerator, threshold float32, d dashboard.Dashboard, c core.Clock, tx *sql.Tx) error {
	now := c.Now()

	kind := gen.kind

	lastExecTime, err := core.RetrieveLastDetectorExecution(tx, kind)

	if err != nil {
		return errorutil.Wrap(err)
	}

	// a similar notification already exists in the past three days, an arbitrary time
	// do not create an insight to it
	if !lastExecTime.IsZero() && now.Sub(lastExecTime) < gen.minimalNotiticationInterval {
		if err := core.StoreLastDetectorExecution(tx, kind, now); err != nil {
			return errorutil.Wrap(err)
		}

		return nil
	}

	interval := timeutil.TimeInterval{From: now.Add(gen.checkTimespan * -1), To: now}

	pairs, err := d.DeliveryStatus(ctx, interval)

	if err != nil {
		return errorutil.Wrap(err)
	}

	total := 0
	bounced := 0

	for _, pair := range pairs {
		//nolint:forcetypeassert
		v := pair.Value.(int)
		total += v

		if pair.Key == "bounced" {
			bounced = v
		}
	}

	value := func() float32 {
		if total == 0 {
			return 0
		}

		return float32(bounced) / float32(total)
	}()

	if value > threshold {
		gen.generate(interval, value)
	}

	if err := core.StoreLastDetectorExecution(tx, kind, now); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func (d Detector) Step(c core.Clock, tx *sql.Tx) error {
	ctx := context.Background()

	for _, g := range d.generators {
		if err := tryToDetectAndGenerateInsight(ctx, g, d.bounceRateThreshold, d.dashboard, c, tx); err != nil {
			return errorutil.Wrap(err)
		}

		if err := g.Step(c, tx); err != nil {
			return errorutil.Wrap(err)
		}
	}

	return nil
}

type BounceRateContent struct {
	Value    float32               `json:"value"`
	Interval timeutil.TimeInterval `json:"interval"`
}

func (c BounceRateContent) Title() notificationCore.ContentComponent {
	return &title{}
}

func (c BounceRateContent) Description() notificationCore.ContentComponent {
	return &description{c}
}

func (c BounceRateContent) Metadata() notificationCore.ContentMetadata {
	return nil
}

type title struct{}

func (t title) String() string {
	return translator.Stringfy(t)
}

func (title) TplString() string {
	return translator.I18n("High Bounce Rate")
}

func (title) Args() []interface{} {
	return nil
}

type description struct {
	c BounceRateContent
}

func (d description) String() string {
	return translator.Stringfy(d)
}

func (d description) TplString() string {
	return translator.I18n("%v percent bounce rate between %v and %v")
}

func (d description) Args() []interface{} {
	return []interface{}{int(d.c.Value * 100), d.c.Interval.From, d.c.Interval.To}
}

func (c BounceRateContent) HelpLink(urlContainer core.URLContainer) string {
	return urlContainer.Get(HighBaseBounceRateContentType)
}

func generateInsight(tx *sql.Tx, c core.Clock, creator core.Creator, content BounceRateContent) error {
	properties := core.InsightProperties{
		Time:        c.Now(),
		Category:    core.LocalCategory,
		Rating:      core.BadRating,
		ContentType: HighBaseBounceRateContentType,
		Content:     content,
	}

	if err := creator.GenerateInsight(context.Background(), tx, properties); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func init() {
	core.RegisterContentType(HighBaseBounceRateContentType, HighBaseBounceRateContentTypeId, core.DefaultContentTypeDecoder(&BounceRateContent{}))
}
