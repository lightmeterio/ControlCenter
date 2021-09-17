// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package dbconn

import (
	"context"
	"database/sql"
	"github.com/rs/zerolog/log"
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

// Execute some coded in a transaction
func (conn *RwConn) Tx(f func(*sql.Tx) error) error {
	tx, err := conn.Begin()

	if err != nil {
		return errorutil.Wrap(err)
	}

	if err := f(tx); err != nil {
		if err != nil {
			if err := tx.Rollback(); err != nil {
				return errorutil.Wrap(err)
			}

			return errorutil.Wrap(err)
		}
	}

	if err := tx.Commit(); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
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
	stmts   map[interface{}]*sql.Stmt
}

func (c *RoPooledConn) SetStmt(key interface{}, stmt *sql.Stmt) {
	c.stmts[key] = stmt
	c.Closers.Add(stmt)
}

// GetStmt gets an prepared statement by a key, where the calles does **NOT** own the returned value
func (c *RoPooledConn) GetStmt(key interface{}) *sql.Stmt {
	stmt, ok := c.stmts[key]
	if !ok {
		log.Panic().Msgf("Sql stmt with key %v not implemented!!!!", key)
	}

	return stmt
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
	conn, release, _ := p.AcquireContext(context.Background())
	return conn, release
}

func (p *RoPool) AcquireContext(ctx context.Context) (*RoPooledConn, func(), error) {
	select {
	case c := <-p.pool:
		return c, func() { p.pool <- c }, nil
	case <-ctx.Done():
		return nil, func() {}, errorutil.Wrap(ctx.Err())
	}
}

type PooledPair struct {
	closeutil.Closers

	RwConn     RwConn
	RoConnPool *RoPool
	Filename   string
}

func Open(filename string, poolSize int) (*PooledPair, error) {
	writer, err := sql.Open("lm_sqlite3", `file:`+filename+`?mode=rwc&cache=private&_loc=auto&_journal=WAL&_sync=OFF&_mutex=no`)

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
		reader, err := sql.Open("lm_sqlite3", `file:`+filename+`?mode=ro&cache=private&_query_only=true&_loc=auto&_journal=WAL&_sync=OFF&_mutex=no`)

		if err != nil {
			return nil, errorutil.Wrap(err)
		}

		conn := &RoPooledConn{
			RoConn:  Ro(reader),
			LocalId: i,
			stmts:   map[interface{}]*sql.Stmt{},
			Closers: closeutil.New(newConnCloser(filename, ROMode, reader)),
		}

		pool.conns = append(pool.conns, conn)
		pool.Closers.Add(conn)

		pool.pool <- conn
	}

	return &PooledPair{RwConn: Rw(writer), RoConnPool: pool, Closers: closeutil.New(newConnCloser(filename, RWMode, writer), pool), Filename: filename}, nil
}
