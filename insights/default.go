package insights

import (
	"gitlab.com/lightmeter/controlcenter/insights/core"
	"gitlab.com/lightmeter/controlcenter/insights/highrate"
	"gitlab.com/lightmeter/controlcenter/insights/mailinactivity"
	"gitlab.com/lightmeter/controlcenter/notification"
)

func NewEngine(
	workspaceDir string,
	notificationCenter notification.Center,
	options core.Options,
) (*Engine, error) {
	return NewCustomEngine(workspaceDir, notificationCenter, options, func(creator *creator, options core.Options) []core.Detector {
		return []core.Detector{
			highrate.NewDetector(creator, options),
			mailinactivity.NewDetector(creator, options),
		}
	})
}
