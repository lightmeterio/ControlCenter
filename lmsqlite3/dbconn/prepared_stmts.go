// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package dbconn

import (
	"database/sql"
	"gitlab.com/lightmeter/controlcenter/pkg/closers"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
)

// Cache statements of a connection, to reuse them over several transactions
type PreparedStmts struct {
	closers.Closers
	stmts []*sql.Stmt
}

// Those are statements created for a given transaction from prepared ones
type TxPreparedStmts struct {
	PreparedStmts
}

// Get obtains the *sql.Stmt in an index.
// Important: the caller does NOT own the returned value
// so you MUST silent the sqlclosecheck linter when using it.
// the statements are closed when TxPreparedStmts.Close() is called,
// just before a transaction is committed or rolled-out
func (s TxPreparedStmts) Get(index int) *sql.Stmt {
	return s.stmts[index]
}

func TxStmts(tx *sql.Tx, stmts PreparedStmts) TxPreparedStmts {
	r := TxPreparedStmts{
		PreparedStmts: BuildPreparedStmts(len(stmts.stmts)),
	}

	// TODO: maybe lazy initialize stmts, as not all of them are always used?
	for i, s := range stmts.stmts {
		txStmt := tx.Stmt(s)
		r.stmts[i] = txStmt
		r.Closers.Add(txStmt)
	}

	return r
}

type StmtsText map[int]string

func BuildPreparedStmts(size int) PreparedStmts {
	return PreparedStmts{stmts: make([]*sql.Stmt, size), Closers: closers.New()}
}

func PrepareRwStmts(stmtsText StmtsText, conn RwConn, stmts *PreparedStmts) error {
	for k, v := range stmtsText {
		//nolint:sqlclosecheck
		stmt, err := conn.Prepare(v)
		if err != nil {
			return errorutil.Wrap(err)
		}

		stmts.stmts[k] = stmt
		stmts.Closers.Add(stmt)
	}

	return nil
}
