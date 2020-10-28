package workspace

import (
	"gitlab.com/lightmeter/controlcenter/dashboard"
	insightscore "gitlab.com/lightmeter/controlcenter/insights/core"
	highrateinsight "gitlab.com/lightmeter/controlcenter/insights/highrate"
	localrblinsight "gitlab.com/lightmeter/controlcenter/insights/localrbl"
	mailinactivityinsight "gitlab.com/lightmeter/controlcenter/insights/mailinactivity"
	messagerblinsight "gitlab.com/lightmeter/controlcenter/insights/messagerbl"
	"gitlab.com/lightmeter/controlcenter/localrbl"
	"gitlab.com/lightmeter/controlcenter/messagerbl"
	"time"
)

const (
	// Those are rough times. They don't need to be so precise to consider leap seconds, and so on...
	oneDay  = time.Hour * 24
	oneWeek = oneDay * 7
)

func insightsOptions(dashboard dashboard.Dashboard, rblChecker localrbl.Checker, rblDetector messagerbl.Stepper) insightscore.Options {
	return insightscore.Options{
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
	}
}
