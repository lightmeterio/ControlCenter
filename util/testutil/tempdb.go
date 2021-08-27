// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package testutil

import (
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"testing"
)

// TempDatabases creates a temporary directory and initialises all databases in it.
// When done, the caller has to execute the returned function to close databases and clear directory.

func TempDatabases(t *testing.T) (dir string, closeDatabases func()) {
	dir, clearDir := TempDir(t)

	err := dbconn.InitialiseDatabasesWithWorkspace(dir)
	if err != nil {
		panic("Could not initialise databases in temp folder " + dir)
	}

	return dir, func() {
		dbconn.DatabasesCloser{}.Close()
		clearDir()
	}
}
