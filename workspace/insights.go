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

func insightsOptions(dashboard dashboard.Dashboard, rblChecker localrbl.Checker, rblDetector messagerbl.Stepper) insightscore.Options {
	return insightscore.Options{
		"dashboard":      dashboard,
		"highrate":       highrateinsight.Options{BaseBounceRateThreshold: 0.3},
		"mailinactivity": mailinactivityinsight.Options{LookupRange: time.Hour * 24, MinTimeGenerationInterval: time.Hour * 12},

		"localrbl": localrblinsight.Options{
			CheckInterval:               time.Hour * 3,
			Checker:                     rblChecker,
			RetryOnScanErrorInterval:    time.Second * 30,
			MinTimeToGenerateNewInsight: time.Hour * 24 * 7,
		},

		"messagerbl": messagerblinsight.Options{
			Detector:                    rblDetector,
			MinTimeToGenerateNewInsight: (time.Hour * 24 * 7) / 2,
		},
	}
}
