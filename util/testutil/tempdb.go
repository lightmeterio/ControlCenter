// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package testutil

import (
	"github.com/rs/zerolog/log"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"path"
	"testing"
)

// TempDBConnection creates new database connection pair in a temporary directory
// It's responsibility of the caller to close the connection before the directory
// is removed.
// basename is an optional value specifying the name for the database file,
// without extension.
func TempDBConnection(t *testing.T, basename ...string) (conn dbconn.ConnPair, removeDir func()) {
	dir, removeDir := TempDir(t)

	filename := path.Join(dir, func() string {
		if len(basename) > 0 {
			return basename[0] + ".db"
		}

		return "database.db"
	}())

	conn, err := dbconn.NewConnPair(filename)

	if err != nil {
		log.Panic().Err(err).Msgf("Error creating temporary database")
	}

	return conn, removeDir
}
