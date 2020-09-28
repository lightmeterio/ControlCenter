package subcommand

import (
	"gitlab.com/lightmeter/controlcenter/util/dbutil"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"log"
)

// The downgrade of multiple database is disallowed to prevent to many failures
func PerformMigrateDownTo(verbose bool, workspaceDirectory, databaseName string, version int64) {
	if workspaceDirectory == "" {
		errorutil.Die(verbose, nil, "No workspace dir specified! Use -help to more info.")
	}

	if databaseName == "" {
		errorutil.Die(verbose, nil, "No database name specified! Use -help to more info.")
	}

	if version == -1 {
		errorutil.Die(verbose, nil, "No migration version specified! Use -help to more info.")
	}

	err := dbutil.MigratorRunDown(workspaceDirectory, databaseName, version)
	if err != nil {
		errorutil.Die(verbose, errorutil.Wrap(err), "Error ", databaseName, " migrate down to version")
	}

	log.Println("migrated database down to version for ", databaseName, " successfully")
}
