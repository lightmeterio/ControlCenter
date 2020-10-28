package welcome

import (
	"database/sql"
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

func NewDetector(creator core.Creator) core.Detector {
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

type content struct{}

func (c content) String() string {
	return ""
}

func (c content) Args() []interface{} {
	return nil
}

func (c content) TplString() string {
	return ""
}

func (d *detector) Step(c core.Clock, tx *sql.Tx) error {
	now := c.Now()

	if err := tryToGenerateWelcomeInsight(d, tx, "welcome", core.InsightProperties{
		Time:        now,
		Category:    core.NewsCategory,
		Content:     content{},
		ContentType: "welcome_content",
		Rating:      core.Unrated,
	}, now); err != nil {
		return errorutil.Wrap(err)
	}

	if err := tryToGenerateWelcomeInsight(d, tx, "insights_introduction", core.InsightProperties{
		Time:        now,
		Category:    core.NewsCategory,
		Content:     content{},
		ContentType: "insights_introduction_content",
		Rating:      core.Unrated,
	}, now); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func init() {
	core.RegisterContentType("welcome_content", 2, core.DefaultContentTypeDecoder(&content{}))
	core.RegisterContentType("insights_introduction_content", 3, core.DefaultContentTypeDecoder(&content{}))
}
