// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package core

import (
	"database/sql"
	notificationCore "gitlab.com/lightmeter/controlcenter/notification/core"
	"gitlab.com/lightmeter/controlcenter/util/closeutil"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
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
	Detectors []Detector
	closers   closeutil.Closers
}

func New(detectors []Detector) (*Core, error) {
	Detectors := []Detector{}
	closers := closeutil.New()

	for _, d := range detectors {
		Detectors = append(Detectors, d)
		closers.Add(d)
	}

	return &Core{
		Detectors: Detectors,
		closers:   closers,
	}, nil
}

func (c *Core) Close() error {
	if err := c.closers.Close(); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
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
