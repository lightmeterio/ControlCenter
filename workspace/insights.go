package workspace

import (
	"gitlab.com/lightmeter/controlcenter/dashboard"
	insightscore "gitlab.com/lightmeter/controlcenter/insights/core"
	highrateinsight "gitlab.com/lightmeter/controlcenter/insights/highrate"
	"gitlab.com/lightmeter/controlcenter/insights/localrbl"
	mailinactivityinsight "gitlab.com/lightmeter/controlcenter/insights/mailinactivity"
	"gitlab.com/lightmeter/controlcenter/localrbl"
	"time"
)

func insightsOptions(dashboard dashboard.Dashboard, rblChecker localrbl.Checker) insightscore.Options {
	return insightscore.Options{
		"dashboard":      dashboard,
		"highrate":       highrateinsight.Options{BaseBounceRateThreshold: 0.3},
		"mailinactivity": mailinactivityinsight.Options{LookupRange: time.Hour * 24, MinTimeGenerationInterval: time.Hour * 12},

		"localrbl": localrblinsight.Options{
			CheckInterval:            time.Hour * 3,
			Checker:                  rblChecker,
			RetryOnScanErrorInterval: time.Second * 30,
		},
	}
}
