// +build !dev release

package insights

import (
	"gitlab.com/lightmeter/controlcenter/insights/core"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
)

func executeAdditionalDetectorsInitialActions([]core.Detector, dbconn.RwConn) error {
	// Intentionally empty as this function intends to be used only development
	return nil
}
