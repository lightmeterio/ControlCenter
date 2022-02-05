// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package core

import (
	"database/sql"
	notificationCore "gitlab.com/lightmeter/controlcenter/notification/core"
	"gitlab.com/lightmeter/controlcenter/pkg/closers"
	insightsSettings "gitlab.com/lightmeter/controlcenter/settings/insights"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"io"
)

type Clock = timeutil.Clock

type Detector interface {
	io.Closer
	Step(Clock, *sql.Tx) error
}

type HistoricalDetector interface {
	Detector
	IsHistoricalDetector()
}

type DetectorWithSettings interface {
	Detector
	UpdateOptionsFromSettings(*insightsSettings.Settings)
}

type Core struct {
	closers.Closers
	Detectors []Detector
}

func New(detectors []Detector) (*Core, error) {
	core := &Core{
		Detectors: []Detector{},
		Closers:   closers.New(),
	}

	for _, d := range detectors {
		core.Detectors = append(core.Detectors, d)
		core.Closers.Add(d)
	}

	return core, nil
}

func (c *Core) UpdateDetectorsFromSettings(settings *insightsSettings.Settings) {
	for _, d := range c.Detectors {
		dws, ok := d.(DetectorWithSettings)
		if !ok {
			continue
		}

		dws.UpdateOptionsFromSettings(settings)
	}
}

type Content interface {
	notificationCore.Content
}

type URLContainer interface {
	Get(k string) string
}

type RecommendationHelpLinkProvider interface {
	HelpLink(container URLContainer) string
}
