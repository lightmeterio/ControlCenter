package subcommand

import (
	"context"
	"gitlab.com/lightmeter/controlcenter/auth"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"log"
	"time"
)

func PerformPasswordReset(verbose bool, workspaceDirectory, emailToReset, passwordToReset string) {
	auth, err := auth.NewAuth(workspaceDirectory, auth.Options{})

	if err != nil {
		errorutil.Die(verbose, errorutil.Wrap(err), "Error opening auth database:", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)

	defer cancel()

	if err := auth.ChangePassword(ctx, emailToReset, passwordToReset); err != nil {
		errorutil.Die(verbose, errorutil.Wrap(err), "Error resetting password:", err)
	}

	if err := auth.Close(); err != nil {
		errorutil.Die(verbose, errorutil.Wrap(err), "Error closing auth database:", err)
	}

	log.Println("Password for user", emailToReset, "reset successfully")
}
