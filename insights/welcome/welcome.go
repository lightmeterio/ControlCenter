// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package welcome

import (
	"context"
	"database/sql"
	"gitlab.com/lightmeter/controlcenter/insights/core"
	notificationCore "gitlab.com/lightmeter/controlcenter/notification/core"
	insightsSettings "gitlab.com/lightmeter/controlcenter/settings/insights"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"time"
)

type detector struct {
	creator core.Creator
}

func (*detector) Close() error {
	return nil
}

func (d *detector) UpdateOptionsFromSettings(settings *insightsSettings.Settings) {}

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

	if err := d.creator.GenerateInsight(context.Background(), tx, properties); err != nil {
		return errorutil.Wrap(err)
	}

	if err := core.StoreLastDetectorExecution(tx, kind, now); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

type emptyComponent struct{}

func (emptyComponent) String() string {
	return ""
}

func (emptyComponent) TplString() string {
	return ""
}

func (emptyComponent) Args() []interface{} {
	return nil
}

type content struct{}

func (content) Title() notificationCore.ContentComponent {
	return &emptyComponent{}
}

func (content) Description() notificationCore.ContentComponent {
	return &emptyComponent{}
}

func (content) Metadata() notificationCore.ContentMetadata {
	return nil
}

func (d *detector) Step(c core.Clock, tx *sql.Tx) error {
	now := c.Now()

	if err := tryToGenerateWelcomeInsight(d, tx, "welcome", core.InsightProperties{
		Time:        now,
		Category:    core.NewsCategory,
		Content:     content{},
		ContentType: WelcomeContentType,
		Rating:      core.Unrated,
	}, now); err != nil {
		return errorutil.Wrap(err)
	}

	if err := tryToGenerateWelcomeInsight(d, tx, "insights_introduction", core.InsightProperties{
		Time:        now,
		Category:    core.NewsCategory,
		Content:     content{},
		ContentType: InsightsIntroductionContentType,
		Rating:      core.Unrated,
	}, now); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

const (
	WelcomeContentType                = "welcome_content"
	WelcomeContentTypeId              = 2
	InsightsIntroductionContentType   = "insights_introduction_content"
	InsightsIntroductionContentTypeId = 3
)

func init() {
	core.RegisterContentType(WelcomeContentType, WelcomeContentTypeId, core.DefaultContentTypeDecoder(&content{}))
	core.RegisterContentType(InsightsIntroductionContentType, InsightsIntroductionContentTypeId, core.DefaultContentTypeDecoder(&content{}))
}
