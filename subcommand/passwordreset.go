// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package subcommand

import (
	"context"
	"github.com/rs/zerolog/log"
	"gitlab.com/lightmeter/controlcenter/auth"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"time"
)

func PerformPasswordReset(verbose bool, workspaceDirectory, emailToReset, passwordToReset string) {
	auth, err := auth.NewAuth(workspaceDirectory, auth.Options{})

	if err != nil {
		errorutil.Dief(verbose, errorutil.Wrap(err), "Error opening auth database")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)

	defer cancel()

	if err := auth.ChangePassword(ctx, emailToReset, passwordToReset); err != nil {
		errorutil.Dief(verbose, errorutil.Wrap(err), "Error resetting password")
	}

	if err := auth.Close(); err != nil {
		errorutil.Dief(verbose, errorutil.Wrap(err), "Error closing auth database")
	}

	log.Info().Msgf("Password for user %s reset successfully", emailToReset)
}
