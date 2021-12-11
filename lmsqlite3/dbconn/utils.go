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

// Execute some code in a transaction
func (conn *RwConn) Tx(ctx context.Context, f func(context.Context, *sql.Tx) error) error {
	tx, err := conn.BeginTx(ctx, nil)

	if err != nil {
		return errorutil.Wrap(err)
	}

	if err := f(ctx, tx); err != nil {
		if err := tx.Rollback(); err != nil {
			return errorutil.Wrap(err)
		}

		return errorutil.Wrap(err)
	}

	if err := tx.Commit(); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

type RoPooledConn struct {
	closeutil.Closers
	RoConn

	LocalId int
	stmts   map[interface{}]*sql.Stmt
}

func (c *RoPooledConn) PrepareStmt(query string, key interface{}) error {
	if _, ok := c.stmts[key]; ok {
		log.Panic().Msgf("A prepared statement for %v already exists!", key)
	}

	stmt, err := c.Prepare(query)
	if err != nil {
		return errorutil.Wrap(err)
	}

	c.stmts[key] = stmt
	c.Closers.Add(stmt)

	return nil
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

func Open(filename string, poolSize int) (pair *PooledPair, err error) {
	writer, err := sql.Open("lm_sqlite3", `file:`+filename+`?mode=rwc&cache=private&_loc=auto&_journal=WAL&_sync=OFF&_mutex=no`)
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	defer func() {
		if err != nil {
			errorutil.UpdateErrorFromCloser(writer, &err)
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
			RoConn:  RoConn{reader},
			LocalId: i,
			stmts:   map[interface{}]*sql.Stmt{},
			Closers: closeutil.New(newConnCloser(filename, ROMode, reader)),
		}

		pool.conns = append(pool.conns, conn)
		pool.Closers.Add(conn)

		pool.pool <- conn
	}

	return &PooledPair{RwConn: RwConn{writer}, RoConnPool: pool, Closers: closeutil.New(newConnCloser(filename, RWMode, writer), pool), Filename: filename}, nil
}
