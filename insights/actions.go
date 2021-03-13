// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package insights

import (
	"database/sql"
	"gitlab.com/lightmeter/controlcenter/insights/core"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
)

// This code is never executed in production, albeit during development,
// adding some sample insights when the application starts, for making tests
// and experimentation easier
//nolint:deadcode,unused
func addInsightsSamples(detectors []core.Detector, conn dbconn.RwConn, clock core.Clock) error {
	if err := conn.Tx(func(tx *sql.Tx) error {
		//nolint:unused
		type sampleInsightGenerator interface {
			GenerateSampleInsight(*sql.Tx, core.Clock) error
		}

		for _, d := range detectors {
			g, canGenerateInsight := d.(sampleInsightGenerator)

			if !canGenerateInsight {
				// it's ok if a generator is not able to generate samples, as it's an optional behaviour
				continue
			}

			if err := g.GenerateSampleInsight(tx, clock); err != nil {
				return errorutil.Wrap(err)
			}
		}

		return nil
	}); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}
