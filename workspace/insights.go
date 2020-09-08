package workspace

import (
	"gitlab.com/lightmeter/controlcenter/dashboard"
	insightscore "gitlab.com/lightmeter/controlcenter/insights/core"
	highrateinsight "gitlab.com/lightmeter/controlcenter/insights/highrate"
	mailinactivityinsight "gitlab.com/lightmeter/controlcenter/insights/mailinactivity"
	"time"
)

func insightsOptions(dashboard dashboard.Dashboard) insightscore.Options {
	return insightscore.Options{
		"dashboard":      dashboard,
		"highrate":       highrateinsight.Options{WeeklyBounceRateThreshold: 0.3},
		"mailinactivity": mailinactivityinsight.Options{LookupRange: time.Hour * 12, MinTimeGenerationInterval: time.Hour * 8},
	}
}
