package mailinactivity

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"gitlab.com/lightmeter/controlcenter/dashboard"
	"gitlab.com/lightmeter/controlcenter/data"
	"gitlab.com/lightmeter/controlcenter/insights/core"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"time"
)

type content struct {
	Interval data.TimeInterval
}

func (c content) String() string {
	return fmt.Sprintf("No emails were sent between %v and %v", c.Interval.From, c.Interval.To)
}

type generator struct {
	creator  core.Creator
	interval *data.TimeInterval
}

func (*generator) Close() error {
	return nil
}

const ContentType = "mail_inactivity"

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

func (g *generator) generate(interval data.TimeInterval) {
	g.interval = &interval
}

type detector struct {
	dashboard dashboard.Dashboard
	options   Options
	creator   core.Creator
	generator *generator
}

func (*detector) Close() error {
	return nil
}

func NewDetector(creator core.Creator, options core.Options) core.Detector {
	d, ok := options["dashboard"].(dashboard.Dashboard)

	if !ok {
		errorutil.MustSucceed(errors.New("Invalid dashboard"), "")
	}

	detectorOptions, ok := options["mailinactivity"].(Options)

	if !ok {
		errorutil.MustSucceed(errors.New("Invalid detector options"), "")
	}

	return &detector{
		dashboard: d,
		options:   detectorOptions,
		creator:   creator,
		generator: &generator{creator: creator, interval: nil},
	}
}

func execChecksForMailInactivity(d *detector, c core.Clock, tx *sql.Tx) error {
	now := c.Now()

	kind := "mail_inactivity"

	lastExecTime, err := core.RetrieveLastDetectorExecution(tx, kind)

	if err != nil {
		return errorutil.Wrap(err)
	}

	interval := data.TimeInterval{
		From: now.Add(-d.options.LookupRange),
		To:   now,
	}

	// no time: first execution, does the check
	// time less than one minute: does nothing
	// time greater than one minute: execute generator

	if !(lastExecTime.IsZero() || (!lastExecTime.IsZero() && now.Sub(lastExecTime) >= d.options.MinTimeGenerationInterval)) {
		return nil
	}

	activityTotalForPair := func(pairs dashboard.Pairs, err error) (int, error) {
		if err != nil {
			return 0, errorutil.Wrap(err)
		}

		total := 0

		for _, pair := range pairs {
			v := pair.Value.(int)
			total += v
		}

		return total, nil
	}

	totalCurrentInterval, err := activityTotalForPair(d.dashboard.DeliveryStatus(interval))

	if err != nil {
		return errorutil.Wrap(err)
	}

	if totalCurrentInterval > 0 {
		return nil
	}

	if lastExecTime.IsZero() {
		// pottentially first insight generation
		totalPreviousInterval, err := activityTotalForPair(d.dashboard.DeliveryStatus(data.TimeInterval{
			From: interval.From.Add(d.options.LookupRange * -1),
			To:   interval.To.Add(d.options.LookupRange * -1),
		}))

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

func (d *detector) Step(c core.Clock, tx *sql.Tx) error {
	if err := execChecksForMailInactivity(d, c, tx); err != nil {
		return errorutil.Wrap(err)
	}

	if err := d.generator.Step(c, tx); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func generateInsight(tx *sql.Tx, c core.Clock, creator core.Creator, interval data.TimeInterval) error {
	properties := core.InsightProperties{
		Time:        c.Now(),
		Category:    core.LocalCategory,
		Rating:      core.BadRating,
		ContentType: ContentType,
		Content: content{
			Interval: interval,
		},
	}

	if err := creator.GenerateInsight(tx, properties); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

// Executed only on development builds, for better developer experience
func (d *detector) GenerateSampleInsight(tx *sql.Tx, c core.Clock) error {
	if err := generateInsight(tx, c, d.creator, data.TimeInterval{From: c.Now().Add(-d.options.LookupRange), To: c.Now()}); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func init() {
	core.RegisterContentType(ContentType, 0, func(b []byte) (core.Content, error) {
		content := content{}
		err := json.Unmarshal(b, &content)

		if err != nil {
			return nil, errorutil.Wrap(err)
		}

		return &content, nil
	})
}
