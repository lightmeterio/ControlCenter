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

	stmts []*sql.Stmt
}

func (s TxPreparedStmts) Get(index uint) *sql.Stmt {
	return s.stmts[index]
}

func (s TxPreparedStmts) Set(index uint, stmt *sql.Stmt) {
	s.stmts[index] = stmt
}

func TxStmts(tx *sql.Tx, stmts PreparedStmts) TxPreparedStmts {
	r := TxPreparedStmts{
		stmts:   make([]*sql.Stmt, len(stmts)),
		Closers: closeutil.New(),
	}

	// TODO: maybe lazy initialize stmts, as not all of them are always used?
	for i, s := range stmts {
		txStmt := tx.Stmt(s)
		r.stmts[i] = txStmt
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
