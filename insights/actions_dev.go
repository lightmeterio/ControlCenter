// +build dev

package insights

import (
	"gitlab.com/lightmeter/controlcenter/insights/core"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
)

func executeAdditionalDetectorsInitialActions(detectors []core.Detector, conn dbconn.RwConn) error {
	// During development, it's useful to have the insights dashboard properly populated with some insights
	// to make testing them (and styling, etc.) easier.
	return addInsightsSamples(detectors, conn)
}
