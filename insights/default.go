// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package insights

import (
	"gitlab.com/lightmeter/controlcenter/insights/blockedips"
	"gitlab.com/lightmeter/controlcenter/insights/blockedipssummary"
	"gitlab.com/lightmeter/controlcenter/insights/core"
	"gitlab.com/lightmeter/controlcenter/insights/detectiveescalation"
	"gitlab.com/lightmeter/controlcenter/insights/highrate"
	"gitlab.com/lightmeter/controlcenter/insights/localrbl"
	"gitlab.com/lightmeter/controlcenter/insights/mailinactivity"
	"gitlab.com/lightmeter/controlcenter/insights/messagerbl"
	"gitlab.com/lightmeter/controlcenter/insights/newsfeed"
	"gitlab.com/lightmeter/controlcenter/insights/welcome"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/metadata"
	"gitlab.com/lightmeter/controlcenter/notification"
	insightsSettings "gitlab.com/lightmeter/controlcenter/settings/insights"
)

func NoDetectors(settings *insightsSettings.Settings, creator *creator, options core.Options) []core.Detector {
	return []core.Detector{}
}

// SettingsDetectors is a list of detectors that take some of their options from the settings (list is used for unit tests purposes)
func SettingsDetectors(settings *insightsSettings.Settings, creator *creator, options core.Options) []core.Detector {
	return []core.Detector{
		highrate.NewDetector(settings, creator, options),
	}
}

func defaultDetectors(settings *insightsSettings.Settings, creator *creator, options core.Options) []core.Detector {
	return []core.Detector{
		highrate.NewDetector(settings, creator, options),
		mailinactivity.NewDetector(creator, options),
		welcome.NewDetector(creator),
		localrblinsight.NewDetector(creator, options),
		messagerblinsight.NewDetector(creator, options),
		newsfeed.NewDetector(creator, options),
		detectiveescalation.NewDetector(creator, options),
		blockedips.NewDetector(creator, options),
		blockedipssummary.NewDetector(creator, options),
	}
}

var NoAdditionalActions = func([]core.Detector, dbconn.RwConn, core.Clock) error { return nil }

func NewEngine(
	metaReader *metadata.Reader,
	insightsAccessor *Accessor,
	fetcher core.Fetcher,
	notificationCenter *notification.Center,
	options core.Options,
) (*Engine, error) {
	return NewCustomEngine(metaReader, insightsAccessor, fetcher, notificationCenter, options, defaultDetectors, executeAdditionalDetectorsInitialActions)
}
