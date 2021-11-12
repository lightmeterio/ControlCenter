// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package insights

import (
	"gitlab.com/lightmeter/controlcenter/insights/bruteforcesummary"
	"gitlab.com/lightmeter/controlcenter/insights/core"
	"gitlab.com/lightmeter/controlcenter/insights/detectiveescalation"
	"gitlab.com/lightmeter/controlcenter/insights/highrate"
	"gitlab.com/lightmeter/controlcenter/insights/localrbl"
	"gitlab.com/lightmeter/controlcenter/insights/mailinactivity"
	"gitlab.com/lightmeter/controlcenter/insights/messagerbl"
	"gitlab.com/lightmeter/controlcenter/insights/newsfeed"
	"gitlab.com/lightmeter/controlcenter/insights/welcome"
	"gitlab.com/lightmeter/controlcenter/notification"
)

func NoDetectors(creator *creator, options core.Options) []core.Detector {
	return []core.Detector{}
}

func defaultDetectors(creator *creator, options core.Options) []core.Detector {
	return []core.Detector{
		highrate.NewDetector(creator, options),
		mailinactivity.NewDetector(creator, options),
		welcome.NewDetector(creator),
		localrblinsight.NewDetector(creator, options),
		messagerblinsight.NewDetector(creator, options),
		newsfeed.NewDetector(creator, options),
		detectiveescalation.NewDetector(creator, options),
		bruteforcesummary.NewDetector(creator, options),
	}
}

func NewEngine(
	c *Accessor,
	notificationCenter *notification.Center,
	options core.Options,
) (*Engine, error) {
	return NewCustomEngine(c, notificationCenter, options, defaultDetectors, executeAdditionalDetectorsInitialActions)
}
