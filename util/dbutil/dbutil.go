// SPDX-FileCopyrightText: 2020,  Lightmeter <hello@lightmeter.io>
// SPDX-License-Identifier: AGPL-3.0

package dbutil

import (
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/migrator"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"path"
)

func InitConnPair(workspaceDirectory, filename string) (dbconn.ConnPair, func(), error) {
	dbFilename := path.Join(workspaceDirectory, filename)

	connPair, err := dbconn.NewConnPair(dbFilename)
	if err != nil {
		return dbconn.ConnPair{}, nil, errorutil.Wrap(err)
	}

	f := func() {
		errorutil.MustSucceed(connPair.Close(), "Closing connection on error")
	}

	return connPair, f, nil
}

func MigratorRunDown(workspaceDirectory string, databaseName string, version int64) error {
	connPair, closeHandler, err := InitConnPair(workspaceDirectory, databaseName+".db")
	if err != nil {
		return errorutil.Wrap(err)
	}

	defer closeHandler()

	if err := migrator.RunDownTo(connPair.RwConn.DB, databaseName, version); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}
