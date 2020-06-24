package lmsqlite3

/*
 * Any custom function exposed to SQLite should be registred in this file
 */

import (
	"crypto/sha256"
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

func computeSha256Sum(b []byte) []byte {
	hash := sha256.New()
	return hash.Sum(b)
}

func init() {
	sql.Register("lm_sqlite3", &sqlite.SQLiteDriver{
		ConnectHook: func(conn *sqlite.SQLiteConn) error {
			if err := conn.RegisterFunc("lm_ip_to_string", ipToString, true); err != nil {
				return err
			}

			if err := conn.RegisterFunc("lm_sha256_sum", computeSha256Sum, true); err != nil {
				return err
			}
			return nil
		},
	})
}
