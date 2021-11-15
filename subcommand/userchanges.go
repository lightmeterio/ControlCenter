// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package subcommand

import (
	"context"
	"github.com/rs/zerolog/log"
	"gitlab.com/lightmeter/controlcenter/auth"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/migrator"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"os"
	"path"
	"time"
)

func removeAllHTTPSessions(workspaceDirectory string) error {
	sessionsDir := path.Join(workspaceDirectory, "http_sessions")

	entries, err := os.ReadDir(sessionsDir)

	// Does nothing if the direct does not yet exist...
	if err != nil && os.IsNotExist(err) {
		return nil
	}

	if err != nil {
		return errorutil.Wrap(err)
	}

	// Finally, reset all existing sessions
	for _, entry := range entries {
		name := path.Join(sessionsDir, entry.Name())
		if err := os.Remove(name); err != nil {
			return errorutil.Wrap(err)
		}
	}

	return nil
}

func PerformUserInfoChange(workspaceDirectory, email, newEmail, name, password string) {
	connPair, err := dbconn.Open(path.Join(workspaceDirectory, "auth.db"), 10)

	if err != nil {
		errorutil.Dief(errorutil.Wrap(err), "Error opening auth database")
	}

	defer connPair.Close()

	if err := migrator.Run(connPair.RwConn.DB, "auth"); err != nil {
		errorutil.Dief(errorutil.Wrap(err), "Error migrating auth database")
	}

	auth, err := auth.NewAuth(connPair, auth.Options{})

	if err != nil {
		errorutil.Dief(errorutil.Wrap(err), "Error instanciating auth object")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	if err := auth.ChangeUserInfo(ctx, email, newEmail, name, password); err != nil {
		errorutil.Dief(errorutil.Wrap(err), "Error Changing user Info password")
	}

	// Finally, reset all existing sessions
	if err := removeAllHTTPSessions(workspaceDirectory); err != nil {
		errorutil.Dief(errorutil.Wrap(err), "Could not clear current sessions")
	}

	log.Info().Msgf("Info change for user %s successfully performed", email)
}
