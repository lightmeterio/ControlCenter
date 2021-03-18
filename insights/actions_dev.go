// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

// +build dev

package insights

import (
	"gitlab.com/lightmeter/controlcenter/insights/core"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
)

func executeAdditionalDetectorsInitialActions(detectors []core.Detector, conn dbconn.RwConn, clock core.Clock) error {
	// During development, it's useful to have the insights dashboard properly populated with some insights
	// to make testing them (and styling, etc.) easier.
	return addInsightsSamples(detectors, conn, clock)
}
