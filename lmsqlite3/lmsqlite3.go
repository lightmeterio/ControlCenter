package lmsqlite3

/*
 * Any custom function exposed to SQLite should be registred in this file
 */

import (
	"database/sql"
	sqlite "github.com/mattn/go-sqlite3"
	"gitlab.com/lightmeter/controlcenter/util"
	"golang.org/x/crypto/bcrypt"
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

const bcryptCost = bcrypt.DefaultCost

func computeBcryptSum(b []byte) []byte {
	r, err := bcrypt.GenerateFromPassword(b, bcryptCost)
	util.MustSucceed(err, "computing bcrypt")
	return r
}

func compareBcryptValue(hash, v []byte) bool {
	return bcrypt.CompareHashAndPassword(hash, v) == nil
}

func init() {
	sql.Register("lm_sqlite3", &sqlite.SQLiteDriver{
		ConnectHook: func(conn *sqlite.SQLiteConn) error {
			if err := conn.RegisterFunc("lm_ip_to_string", ipToString, true); err != nil {
				return err
			}

			if err := conn.RegisterFunc("lm_bcrypt_sum", computeBcryptSum, true); err != nil {
				return err
			}

			if err := conn.RegisterFunc("lm_bcrypt_compare", compareBcryptValue, true); err != nil {
				return err
			}

			return nil
		},
	})
}
