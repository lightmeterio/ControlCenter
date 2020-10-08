package testutil

import (
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"log"
	"path"
)

// TempDBConnection creates new database connection pair in a temporary directory
// It's responsibility of the caller to close the connection before the directory
// is removed
func TempDBConnection() (conn dbconn.ConnPair, removeDir func()) {
	dir, removeDir := TempDir()
	conn, err := dbconn.NewConnPair(path.Join(dir, "database.db"))

	if err != nil {
		log.Panicln("Error creating temporary database:", err)
	}

	return conn, removeDir
}
