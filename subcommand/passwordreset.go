package subcommand

import (
	"gitlab.com/lightmeter/controlcenter/auth"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"log"
)

func PerformPasswordReset(verbose bool, workspaceDirectory, emailToReset, passwordToReset string) {
	auth, err := auth.NewAuth(workspaceDirectory, auth.Options{})

	if err != nil {
		errorutil.Die(verbose, errorutil.Wrap(err), "Error opening auth database:", err)
	}

	if err := auth.ChangePassword(emailToReset, passwordToReset); err != nil {
		errorutil.Die(verbose, errorutil.Wrap(err), "Error resetting password:", err)
	}

	if err := auth.Close(); err != nil {
		errorutil.Die(verbose, errorutil.Wrap(err), "Error closing auth database:", err)
	}

	log.Println("Password for user", emailToReset, "reset successfully")
}
