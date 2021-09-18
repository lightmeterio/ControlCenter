// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package dbconn

import (
	"database/sql"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
)

type PreparedStmts []*sql.Stmt

type StmtsText map[uint]string

func (stmts PreparedStmts) Close() error {
	for _, stmt := range stmts {
		if err := stmt.Close(); err != nil {
			return errorutil.Wrap(err)
		}
	}

	return nil
}

func PrepareRwStmts(stmtsText StmtsText, conn RwConn, stmts PreparedStmts) error {
	for k, v := range stmtsText {
		//nolint:sqlclosecheck
		stmt, err := conn.Prepare(v)
		if err != nil {
			return errorutil.Wrap(err)
		}

		stmts[k] = stmt
	}

	return nil
}
