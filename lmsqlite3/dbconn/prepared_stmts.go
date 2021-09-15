// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package dbconn

import (
	"database/sql"
	"gitlab.com/lightmeter/controlcenter/util/closeutil"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
)

type PreparedStmts []*sql.Stmt

// Those are statements created from a Tx, ready to be used
type TxPreparedStmts struct {
	closeutil.Closers

	S []*sql.Stmt
}

func TxStmts(tx *sql.Tx, stmts PreparedStmts) TxPreparedStmts {
	r := TxPreparedStmts{
		S:       make([]*sql.Stmt, len(stmts)),
		Closers: closeutil.New(),
	}

	for i, s := range stmts {
		txStmt := tx.Stmt(s)
		r.S[i] = txStmt
		r.Closers.Add(txStmt)
	}

	return r
}

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
