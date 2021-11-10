// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package insights

import (
	"context"
	"database/sql"
	"gitlab.com/lightmeter/controlcenter/insights/core"
	"gitlab.com/lightmeter/controlcenter/insights/importsummary"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
)

// This code is never executed in production, albeit during development,
// adding some sample insights when the application starts, for making tests
// and experimentation easier
//nolint:deadcode,unused
func addInsightsSamples(detectors []core.Detector, conn dbconn.RwConn, clock core.Clock) error {
	gen := func(detectors []core.Detector, conn dbconn.RwConn, clock core.Clock) error {
		if err := conn.Tx(context.Background(), func(ctx context.Context, tx *sql.Tx) error {
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

	if err := gen(detectors, conn, clock); err != nil {
		return errorutil.Wrap(err)
	}

	// the import summary should not be listed as a normal insight, so it has to be added manually here...
	if err := gen([]core.Detector{importsummary.NewDetector(nil, timeutil.TimeInterval{})}, conn, clock); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}
