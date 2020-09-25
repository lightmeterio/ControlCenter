package workspace

import (
	"gitlab.com/lightmeter/controlcenter/dashboard"
	insightscore "gitlab.com/lightmeter/controlcenter/insights/core"
	highrateinsight "gitlab.com/lightmeter/controlcenter/insights/highrate"
	localrblinsight "gitlab.com/lightmeter/controlcenter/insights/localrbl"
	mailinactivityinsight "gitlab.com/lightmeter/controlcenter/insights/mailinactivity"
	"net"
	"time"
)

func insightsOptions(dashboard dashboard.Dashboard) insightscore.Options {
	return insightscore.Options{
		"dashboard":      dashboard,
		"highrate":       highrateinsight.Options{BaseBounceRateThreshold: 0.3},
		"mailinactivity": mailinactivityinsight.Options{LookupRange: time.Hour * 24, MinTimeGenerationInterval: time.Hour * 12},

		"localrbl": localrblinsight.Options{
			CheckInterval:    time.Second * 30,
			RBLProvidersURLs: localrblinsight.DefaultRBLs,
			CheckedAddress:   net.ParseIP("127.0.0.2"),
		},
	}
}
