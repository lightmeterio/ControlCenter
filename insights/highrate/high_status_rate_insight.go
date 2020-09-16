package highrate

import (
	"database/sql"
	"encoding/json"
	"errors"
	"gitlab.com/lightmeter/controlcenter/dashboard"
	"gitlab.com/lightmeter/controlcenter/data"
	"gitlab.com/lightmeter/controlcenter/insights/core"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"time"
)

var (
	highWeeklyBounceRateContentType = "high_bounce_rate"
)

type Options struct {
	WeeklyBounceRateThreshold float32
}

type weeklyBounceRateInsightsGenerator struct {
	creator core.Creator
	value   *highWeeklyBounceRateInsightContent
}

func (*weeklyBounceRateInsightsGenerator) Close() error {
	return nil
}

func (g *weeklyBounceRateInsightsGenerator) Step(c core.Clock, tx *sql.Tx) error {
	if g.value == nil {
		return nil
	}

	properties := core.InsightProperties{
		Time:        c.Now(),
		Category:    core.LocalCategory,
		Rating:      core.BadRating,
		ContentType: highWeeklyBounceRateContentType,
		Content:     g.value,
	}

	if err := g.creator.GenerateInsight(tx, properties); err != nil {
		return errorutil.WrapError(err)
	}

	g.value = nil

	return nil
}

func (g *weeklyBounceRateInsightsGenerator) generate(interval data.TimeInterval, value float32) {
	g.value = &highWeeklyBounceRateInsightContent{Value: value, Interval: interval}
}

type highRateDetector struct {
	bounceRateThreshold               float32
	dashboard                         dashboard.Dashboard
	weeklyBounceRateInsightsGenerator *weeklyBounceRateInsightsGenerator
}

func (*highRateDetector) Close() error {
	return nil
}

func NewDetector(creator core.Creator, options core.Options) *highRateDetector {
	d, ok := options["dashboard"].(dashboard.Dashboard)

	if !ok {
		errorutil.MustSucceed(errors.New("Invalid dashboard!"), "")
	}

	detectorOptions, ok := options["highrate"].(Options)

	if !ok {
		errorutil.MustSucceed(errors.New("Invalid Options!"), "")
	}

	return &highRateDetector{
		dashboard:                         d,
		bounceRateThreshold:               detectorOptions.WeeklyBounceRateThreshold,
		weeklyBounceRateInsightsGenerator: &weeklyBounceRateInsightsGenerator{creator: creator},
	}
}

func execWeeklyChecks(d *highRateDetector, c core.Clock, tx *sql.Tx) error {
	now := c.Now()

	kind := "high_weekly_bounce_rate"

	lastExecTime, err := core.RetrieveLastDetectorExecution(tx, kind)

	if err != nil {
		return errorutil.WrapError(err)
	}

	// a similar notification already exists in the past three days, an arbitrary time
	// do not create an insight to it
	if !lastExecTime.IsZero() && now.Sub(lastExecTime) < time.Hour*24*3 {
		if err := core.StoreLastDetectorExecution(tx, kind, now); err != nil {
			return errorutil.WrapError(err)
		}

		return nil
	}

	interval := data.TimeInterval{From: now.Add(time.Hour * 24 * 7 * -1), To: now}

	pairs := d.dashboard.DeliveryStatus(interval)

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

	if value > d.bounceRateThreshold {
		d.weeklyBounceRateInsightsGenerator.generate(interval, value)
	}

	if err := core.StoreLastDetectorExecution(tx, kind, now); err != nil {
		return errorutil.WrapError(err)
	}

	return nil
}

func (d *highRateDetector) Step(c core.Clock, tx *sql.Tx) error {
	return execWeeklyChecks(d, c, tx)
}

func (d *highRateDetector) Setup(tx *sql.Tx) error {
	return nil
}

func (d *highRateDetector) Steppers() []core.Stepper {
	return []core.Stepper{d, d.weeklyBounceRateInsightsGenerator}
}

type highWeeklyBounceRateInsightContent struct {
	Value    float32
	Interval data.TimeInterval
}

func init() {
	core.RegisterContentType(highWeeklyBounceRateContentType, 1, func(b []byte) (interface{}, error) {
		var v highWeeklyBounceRateInsightContent

		if err := json.Unmarshal(b, &v); err != nil {
			return nil, errorutil.WrapError(err)
		}

		return &v, nil
	})
}
