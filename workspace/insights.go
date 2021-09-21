// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package workspace

import (
	"gitlab.com/lightmeter/controlcenter/dashboard"
	"gitlab.com/lightmeter/controlcenter/detective/escalator"
	insightscore "gitlab.com/lightmeter/controlcenter/insights/core"
	"gitlab.com/lightmeter/controlcenter/insights/detectiveescalation"
	highrateinsight "gitlab.com/lightmeter/controlcenter/insights/highrate"
	localrblinsight "gitlab.com/lightmeter/controlcenter/insights/localrbl"
	mailinactivityinsight "gitlab.com/lightmeter/controlcenter/insights/mailinactivity"
	messagerblinsight "gitlab.com/lightmeter/controlcenter/insights/messagerbl"
	newsfeedinsight "gitlab.com/lightmeter/controlcenter/insights/newsfeed"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/localrbl"
	"gitlab.com/lightmeter/controlcenter/messagerbl"
	"time"
)

const (
	// Those are rough times. They don't need to be so precise to consider leap seconds, and so on...
	oneDay  = time.Hour * 24
	oneWeek = oneDay * 7
)

func insightsOptions(
	dashboard dashboard.Dashboard,
	rblChecker localrbl.Checker,
	rblDetector messagerbl.Stepper,
	detectiveEscalator escalator.Stepper,
	deliverydbConnPool *dbconn.RoPool,
) insightscore.Options {
	return insightscore.Options{
		"logsConnPool":   deliverydbConnPool,
		"dashboard":      dashboard,
		"highrate":       highrateinsight.Options{BaseBounceRateThreshold: 0.3},
		"mailinactivity": mailinactivityinsight.Options{LookupRange: time.Hour * 24, MinTimeGenerationInterval: time.Hour * 12},

		"localrbl": localrblinsight.Options{
			CheckInterval:               time.Hour * 3,
			Checker:                     rblChecker,
			RetryOnScanErrorInterval:    time.Second * 30,
			MinTimeToGenerateNewInsight: oneWeek,
		},

		"messagerbl": messagerblinsight.Options{
			Detector:                    rblDetector,
			MinTimeToGenerateNewInsight: oneWeek / 2,
		},

		"newsfeed": newsfeedinsight.Options{
			URL:            "https://lightmeter.io/category/news-insights?feed=rss",
			UpdateInterval: time.Hour * 2,
			RetryTime:      time.Minute * 10,
			TimeLimit:      oneDay * 2,
		},

		"detective": detectiveescalation.Options{
			Escalator: detectiveEscalator,
		},
	}
}
