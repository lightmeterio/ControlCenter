package highrate

import (
	"context"
	"database/sql"
	"errors"
	"gitlab.com/lightmeter/controlcenter/dashboard"
	"gitlab.com/lightmeter/controlcenter/data"
	"gitlab.com/lightmeter/controlcenter/i18n/translator"
	"gitlab.com/lightmeter/controlcenter/insights/core"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"time"
)

var (
	highBaseBounceRateContentType = "high_bounce_rate"
)

type Options struct {
	BaseBounceRateThreshold float32
}

type bounceRateGenerator struct {
	creator                     core.Creator
	value                       *bounceRateContent
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

func (g *bounceRateGenerator) generate(interval data.TimeInterval, value float32) {
	g.value = &bounceRateContent{Value: value, Interval: interval}
}

type highRateDetector struct {
	bounceRateThreshold float32
	dashboard           dashboard.Dashboard
	generators          []*bounceRateGenerator
}

func (*highRateDetector) Close() error {
	return nil
}

func NewDetector(creator core.Creator, options core.Options) core.Detector {
	d, ok := options["dashboard"].(dashboard.Dashboard)

	if !ok {
		errorutil.MustSucceed(errors.New("Invalid dashboard"))
	}

	detectorOptions, ok := options["highrate"].(Options)

	if !ok {
		errorutil.MustSucceed(errors.New("Invalid Options"))
	}

	return &highRateDetector{
		dashboard:           d,
		bounceRateThreshold: detectorOptions.BaseBounceRateThreshold,
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

	interval := data.TimeInterval{From: now.Add(gen.checkTimespan * -1), To: now}

	pairs, err := d.DeliveryStatus(ctx, interval)

	if err != nil {
		return errorutil.Wrap(err)
	}

	total := 0
	bounced := 0

	for _, pair := range pairs {
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

func (d *highRateDetector) Step(c core.Clock, tx *sql.Tx) error {
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

type bounceRateContent struct {
	Value    float32
	Interval data.TimeInterval
}

func (c bounceRateContent) String() string {
	return translator.Stringfy(c)
}

func (c bounceRateContent) TplString() string {
	return translator.I18n("%%v percent bounce rate between %%v and %%v")
}

func (c bounceRateContent) Args() []interface{} {
	return []interface{}{int(c.Value * 100), c.Interval.From, c.Interval.To}
}

func generateInsight(tx *sql.Tx, c core.Clock, creator core.Creator, content bounceRateContent) error {
	properties := core.InsightProperties{
		Time:        c.Now(),
		Category:    core.LocalCategory,
		Rating:      core.BadRating,
		ContentType: highBaseBounceRateContentType,
		Content:     content,
	}

	if err := creator.GenerateInsight(tx, properties); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

// Executed only on development builds, for better developer experience
func (d *highRateDetector) GenerateSampleInsight(tx *sql.Tx, c core.Clock) error {
	for _, g := range d.generators {
		now := c.Now()

		content := bounceRateContent{
			Value:    0.9,
			Interval: data.TimeInterval{From: now.Add(g.checkTimespan * -1), To: now},
		}

		if err := generateInsight(tx, c, g.creator, content); err != nil {
			return errorutil.Wrap(err)
		}
	}

	return nil
}

func init() {
	core.RegisterContentType(highBaseBounceRateContentType, 1, core.DefaultContentTypeDecoder(&bounceRateContent{}))
}
