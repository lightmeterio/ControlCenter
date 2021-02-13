// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package dbconn

import (
	"database/sql"
	_ "gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/util/closeutil"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
)

type RoConn struct {
	*sql.DB
}

type RwConn struct {
	*sql.DB
}

func Ro(db *sql.DB) RoConn {
	return RoConn{db}
}

func Rw(db *sql.DB) RwConn {
	return RwConn{db}
}

type RoPooledConn struct {
	closeutil.Closers
	RoConn

	LocalId int
	Stmts   map[interface{}]*sql.Stmt
}

type RoPool struct {
	closeutil.Closers

	conns []*RoPooledConn
	pool  chan *RoPooledConn
}

func (p *RoPool) ForEach(f func(*RoPooledConn) error) error {
	for _, v := range p.conns {
		if err := f(v); err != nil {
			return errorutil.Wrap(err)
		}
	}

	return nil
}

func (p *RoPool) Acquire() (*RoPooledConn, func()) {
	c := <-p.pool
	return c, func() { p.pool <- c }
}

type PooledPair struct {
	closeutil.Closers

	RwConn     RwConn
	RoConnPool *RoPool
}

func Open(filename string, poolSize int) (*PooledPair, error) {
	writer, err := sql.Open("lm_sqlite3", `file:`+filename+`?mode=rwc&cache=private&_loc=auto&_journal=WAL&_sync=OFF`)

	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	defer func() {
		if err != nil {
			errorutil.MustSucceed(writer.Close(), "Closing RW connection on error")
		}
	}()

	pool := &RoPool{
		pool:    make(chan *RoPooledConn, poolSize),
		Closers: closeutil.New(),
	}

	for i := 0; i < poolSize; i++ {
		reader, err := sql.Open("lm_sqlite3", `file:`+filename+`?mode=ro&cache=private&_query_only=true&_loc=auto&_journal=WAL&_sync=OFF`)

		if err != nil {
			return nil, errorutil.Wrap(err)
		}

		conn := &RoPooledConn{
			RoConn:  Ro(reader),
			LocalId: i,
			Stmts:   map[interface{}]*sql.Stmt{},
			Closers: closeutil.New(),
		}

		pool.conns = append(pool.conns, conn)
		pool.Closers.Add(conn)

		pool.pool <- conn
	}

	return &PooledPair{RwConn: Rw(writer), RoConnPool: pool, Closers: closeutil.New(writer, pool)}, nil
}
