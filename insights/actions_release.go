// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

// +build !dev release

package insights

import (
	"gitlab.com/lightmeter/controlcenter/insights/core"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
)

func executeAdditionalDetectorsInitialActions([]core.Detector, dbconn.RwConn, core.Clock) error {
	// Intentionally empty as this function intends to be used only development
	return nil
}
