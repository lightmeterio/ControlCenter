package insights

import (
	"gitlab.com/lightmeter/controlcenter/insights/core"
	"gitlab.com/lightmeter/controlcenter/insights/highrate"
	"gitlab.com/lightmeter/controlcenter/insights/localrbl"
	"gitlab.com/lightmeter/controlcenter/insights/mailinactivity"
	"gitlab.com/lightmeter/controlcenter/insights/welcome"
	"gitlab.com/lightmeter/controlcenter/notification"
)

func defaultDetectors(creator *creator, options core.Options) []core.Detector {
	return []core.Detector{
		highrate.NewDetector(creator, options),
		mailinactivity.NewDetector(creator, options),
		welcome.NewDetector(creator),
		localrblinsight.NewDetector(creator, options),
	}
}

func NewEngine(
	workspaceDir string,
	notificationCenter notification.Center,
	options core.Options,
) (*Engine, error) {
	return NewCustomEngine(workspaceDir, notificationCenter, options, defaultDetectors, executeAdditionalDetectorsInitialActions)
}
