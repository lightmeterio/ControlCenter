package lmsqlite3

/*
 * Any custom function exposed to SQLite should be registred in this file
 */

import (
	"database/sql"
	sqlite "github.com/mattn/go-sqlite3"
	"net"
)

func ipToString(b []byte) string {
	if len(b) == 0 {
		return ""
	}

	if len(b) != net.IPv4len && len(b) != net.IPv6len {
		// TODO: how to handle errors inside sqlite?
		return ""
	}

	ip := net.IP(b)

	return ip.String()
}

func init() {
	sql.Register("lm_sqlite3", &sqlite.SQLiteDriver{
		ConnectHook: func(conn *sqlite.SQLiteConn) error {
			conn.RegisterFunc("lm_ip_to_string", ipToString, true)
			return nil
		},
	})
}
