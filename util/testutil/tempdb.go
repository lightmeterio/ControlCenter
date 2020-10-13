package testutil

import (
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"log"
	"path"
)

// TempDBConnection creates new database connection pair in a temporary directory
// It's responsibility of the caller to close the connection before the directory
// is removed.
// basename is an optional value specifying the name for the database file,
// without extension.
func TempDBConnection(basename ...string) (conn dbconn.ConnPair, removeDir func()) {
	dir, removeDir := TempDir()

	filename := path.Join(dir, func() string {
		if len(basename) > 0 {
			return basename[0] + ".db"
		}

		return "database.db"
	}())

	conn, err := dbconn.NewConnPair(filename)

	if err != nil {
		log.Panicln("Error creating temporary database:", err)
	}

	return conn, removeDir
}
