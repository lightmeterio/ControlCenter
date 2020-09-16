package dbconn

import (
	"database/sql"
	"fmt"
	_ "gitlab.com/lightmeter/controlcenter/lmsqlite3"
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
	RoConn RoConn
	RwConn RwConn
}

func (c *ConnPair) Close() error {
	readerError := c.RoConn.Close()
	writerError := c.RwConn.Close()

	if writerError == nil {
		if readerError != nil {
			return errorutil.Wrap(readerError)
		}

		// no errors at all
		return nil
	}

	// here we know that writeError != nil

	if readerError == nil {
		return errorutil.Wrap(writerError)
	}

	// Both errors exist. We lose the erorrs, keeping only the message, which is ok for now
	return fmt.Errorf("RW: %v, RO: %v", writerError, readerError)
}

func NewConnPair(filename string) (ConnPair, error) {
	writer, err := sql.Open("lm_sqlite3", `file:`+filename+`?mode=rwc&cache=private&_loc=auto&_journal=WAL`)

	if err != nil {
		return ConnPair{}, errorutil.Wrap(err)
	}

	defer func() {
		if err != nil {
			errorutil.MustSucceed(writer.Close(), "Closing RW connection on error")
		}
	}()

	reader, err := sql.Open("lm_sqlite3", `file:`+filename+`?mode=ro&cache=shared&_query_only=true&_loc=auto&_journal=WAL`)

	if err != nil {
		return ConnPair{}, errorutil.Wrap(err)
	}

	return ConnPair{RoConn: Ro(reader), RwConn: Rw(writer)}, nil
}
