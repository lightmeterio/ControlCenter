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

	properties := core.InsightProperties{
		Time:        c.Now(),
		Category:    core.LocalCategory,
		Rating:      core.BadRating,
		ContentType: highBaseBounceRateContentType,
		Content:     g.value,
	}

	if err := g.creator.GenerateInsight(tx, properties); err != nil {
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

func tryToDetectAndGenerateInsight(gen *bounceRateGenerator, threshold float32, d dashboard.Dashboard, c core.Clock, tx *sql.Tx) error {
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

	pairs := d.DeliveryStatus(interval)

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
	for _, g := range d.generators {
		if err := tryToDetectAndGenerateInsight(g, d.bounceRateThreshold, d.dashboard, c, tx); err != nil {
			return errorutil.Wrap(err)
		}
	}

	return nil
}

func (d *highRateDetector) Steppers() []core.Stepper {
	return []core.Stepper{d, d.generators[0]}
}

type bounceRateContent struct {
	Value    float32
	Interval data.TimeInterval
}

func init() {
	core.RegisterContentType(highBaseBounceRateContentType, 1, func(b []byte) (interface{}, error) {
		var v bounceRateContent

		if err := json.Unmarshal(b, &v); err != nil {
			return nil, errorutil.Wrap(err)
		}

		return &v, nil
	})
}
