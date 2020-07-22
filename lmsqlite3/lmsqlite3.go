package lmsqlite3

/*
 * Any custom function exposed to SQLite should be registred in this file
 */

import (
	"database/sql"
	sqlite "github.com/mattn/go-sqlite3"
	"gitlab.com/lightmeter/controlcenter/domainmapping"
	"gitlab.com/lightmeter/controlcenter/util"
	"golang.org/x/crypto/bcrypt"
	"net"
	"sync"
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

const bcryptCost = 12

func computeBcryptSum(b []byte) []byte {
	r, err := bcrypt.GenerateFromPassword(b, bcryptCost)
	util.MustSucceed(err, "computing bcrypt")
	return r
}

func compareBcryptValue(hash, v []byte) bool {
	return bcrypt.CompareHashAndPassword(hash, v) == nil
}

type Options map[string]interface{}

func registerDomainMapping(conn *sqlite.SQLiteConn, options Options) error {
	mapper, ok := options["domain_mapping"].(*domainmapping.Mapper)

	if !ok {
		m, err := domainmapping.Mapping(domainmapping.RawList{})

		if err != nil {
			return util.WrapError(err)
		}

		mapper = &m
	}

	if err := conn.RegisterFunc("lm_resolve_domain_mapping", func(domain string) string {
		return mapper.Resolve(domain)
	}, true); err != nil {
		return util.WrapError(err)
	}

	return nil
}

var once sync.Once

func Initialize(options Options) {
	once.Do(func() {
		sql.Register("lm_sqlite3", &sqlite.SQLiteDriver{
			ConnectHook: func(conn *sqlite.SQLiteConn) error {
				util.MustSucceed(conn.RegisterFunc("lm_ip_to_string", ipToString, true), "")
				util.MustSucceed(conn.RegisterFunc("lm_bcrypt_sum", computeBcryptSum, true), "")
				util.MustSucceed(conn.RegisterFunc("lm_bcrypt_compare", compareBcryptValue, true), "")
				util.MustSucceed(registerDomainMapping(conn, options), "")

				return nil
			},
		})
	})
}
