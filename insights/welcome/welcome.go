package welcome

import (
	"database/sql"
	"encoding/json"
	"gitlab.com/lightmeter/controlcenter/insights/core"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"time"
)

type detector struct {
	creator core.Creator
}

func (*detector) Close() error {
	return nil
}

func NewDetector(creator core.Creator) *detector {
	return &detector{creator}
}

func tryToGenerateWelcomeInsight(d *detector, tx *sql.Tx, kind string, properties core.InsightProperties, now time.Time) error {
	lastExecTime, err := core.RetrieveLastDetectorExecution(tx, kind)

	if err != nil {
		return errorutil.Wrap(err)
	}

	if !lastExecTime.IsZero() {
		return nil
	}

	if err := d.creator.GenerateInsight(tx, properties); err != nil {
		return errorutil.Wrap(err)
	}

	if err := core.StoreLastDetectorExecution(tx, kind, now); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func (d *detector) Step(c core.Clock, tx *sql.Tx) error {
	now := c.Now()

	if err := tryToGenerateWelcomeInsight(d, tx, "welcome", core.InsightProperties{
		Time:        now,
		Category:    core.NewsCategory,
		Content:     struct{}{},
		ContentType: "welcome_content",
		Rating:      core.Unrated,
	}, now); err != nil {
		return errorutil.Wrap(err)
	}

	if err := tryToGenerateWelcomeInsight(d, tx, "insights_introduction", core.InsightProperties{
		Time:        now,
		Category:    core.NewsCategory,
		Content:     struct{}{},
		ContentType: "insights_introduction_content",
		Rating:      core.Unrated,
	}, now); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func (d *detector) Setup(tx *sql.Tx) error {
	return nil
}

func (d *detector) Steppers() []core.Stepper {
	return []core.Stepper{d}
}

func init() {
	handler := func(b []byte) (interface{}, error) {
		content := struct{}{}
		err := json.Unmarshal(b, &content)

		if err != nil {
			return nil, errorutil.Wrap(err)
		}

		return &content, nil
	}

	core.RegisterContentType("welcome_content", 2, handler)
	core.RegisterContentType("insights_introduction_content", 3, handler)
}
