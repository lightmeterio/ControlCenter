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
func addInsightsSamples(detectors []core.Detector, conn dbconn.RwConn) error {
	tx, err := conn.Begin()

	if err != nil {
		return errorutil.Wrap(err)
	}

	defer func() {
		if err != nil {
			errorutil.MustSucceed(tx.Rollback(), "")
		}
	}()

	clock := &realClock{}

	//nolint:unused
	type sampleInsightGenerator interface {
		GenerateSampleInsight(*sql.Tx, core.Clock) error
	}

	for _, d := range detectors {
		g, ok := d.(sampleInsightGenerator)

		if !ok {
			continue
		}

		err = g.GenerateSampleInsight(tx, clock)

		if err != nil {
			return errorutil.Wrap(err)
		}
	}

	err = tx.Commit()

	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}
