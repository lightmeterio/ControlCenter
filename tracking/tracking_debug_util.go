// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

//go:build tracking_debug_util
// +build tracking_debug_util

package tracking

import (
	"database/sql"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
)

func debugTrackingAction(tx **sql.Tx, t *Tracker, batchId *int64, trackerStmts dbconn.PreparedStmts, txStmts *dbconn.TxPreparedStmts) error {
	var err error

	if *tx, err = ensureMessagesArePersistedAndDispatchResults(*tx, t, *batchId, trackerStmts, txStmts); err != nil {
		return errorutil.Wrap(err)
	}

	*batchId++

	return nil
}
