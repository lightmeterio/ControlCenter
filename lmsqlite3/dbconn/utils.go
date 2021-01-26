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

type ConnPair struct {
	RoConn  RoConn
	RwConn  RwConn
	closers closeutil.Closers
}

func (c *ConnPair) Close() error {
	if err := c.closers.Close(); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func NewConnPair(filename string) (ConnPair, error) {
	writer, err := sql.Open("lm_sqlite3", `file:`+filename+`?mode=rwc&cache=private&_loc=auto&_journal=WAL&_sync=OFF`)

	if err != nil {
		return ConnPair{}, errorutil.Wrap(err)
	}

	defer func() {
		if err != nil {
			errorutil.MustSucceed(writer.Close(), "Closing RW connection on error")
		}
	}()

	reader, err := sql.Open("lm_sqlite3", `file:`+filename+`?mode=ro&cache=shared&_query_only=true&_loc=auto&_journal=WAL&_sync=OFF`)

	if err != nil {
		return ConnPair{}, errorutil.Wrap(err)
	}

	return ConnPair{RoConn: Ro(reader), RwConn: Rw(writer), closers: closeutil.New(reader, writer)}, nil
}
