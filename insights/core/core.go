// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package core

import (
	"database/sql"
	notificationCore "gitlab.com/lightmeter/controlcenter/notification/core"
	"gitlab.com/lightmeter/controlcenter/util/closeutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
)

type Clock = timeutil.Clock

type Detector interface {
	Step(Clock, *sql.Tx) error
	Close() error
}

type HistoricalDetector interface {
	Detector
	IsHistoricalDetector()
}

type Core struct {
	closeutil.Closers
	Detectors []Detector
}

func New(detectors []Detector) (*Core, error) {
	core := &Core{
		Detectors: []Detector{},
		Closers:   closeutil.New(),
	}

	for _, d := range detectors {
		core.Detectors = append(core.Detectors, d)
		core.Closers.Add(d)
	}

	return core, nil
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
