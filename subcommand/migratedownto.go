// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package subcommand

import (
	"github.com/rs/zerolog/log"
	"gitlab.com/lightmeter/controlcenter/util/dbutil"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
)

// The downgrade of multiple database is disallowed to prevent to many failures
func PerformMigrateDownTo(workspaceDirectory, databaseName string, version int64) {
	if workspaceDirectory == "" {
		errorutil.Dief(nil, "No workspace dir specified! Use -help to more info.")
	}

	if databaseName == "" {
		errorutil.Dief(nil, "No database name specified! Use -help to more info.")
	}

	if version == -1 {
		errorutil.Dief(nil, "No migration version specified! Use -help to more info.")
	}

	err := dbutil.MigratorRunDown(workspaceDirectory, databaseName, version)
	if err != nil {
		errorutil.Dief(errorutil.Wrap(err), "Error %s migrate down to version", databaseName)
	}

	log.Info().Msgf("migrated database down to version for %v successfully", databaseName)
}
