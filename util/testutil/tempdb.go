// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package testutil

import (
	"github.com/rs/zerolog/log"
	//nolint:golint
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/migrator"
	"path"
	"testing"
)

// TempDBConnection creates new database connection pair in a temporary directory
// It's responsibility of the caller to close the connection before the directory
// is removed.
// basename is an optional value specifying the name for the database file,
// without extension.
func TempDBConnection(t *testing.T, basename ...string) (*dbconn.PooledPair, func()) {
	dir, removeDir := TempDir(t)

	filename := path.Join(dir, func() string {
		if len(basename) > 0 {
			return basename[0] + ".db"
		}

		return "database.db"
	}())

	conn, err := dbconn.Open(filename, 5)
	So(err, ShouldBeNil)

	if err != nil {
		log.Panic().Err(err).Msgf("Error creating temporary database")
	}

	return conn, func() {
		So(conn.Close(), ShouldBeNil)
		removeDir()
	}
}

func TempDBConnectionMigrated(t *testing.T, databaseName string) (*dbconn.PooledPair, func()) {
	conn, removeDir := TempDBConnection(t, databaseName)

	if err := migrator.Run(conn.RwConn.DB, databaseName); err != nil {
		log.Panic().Err(err).Msgf("Error migrating temporary database %s", databaseName)
	}

	return conn, removeDir
}
