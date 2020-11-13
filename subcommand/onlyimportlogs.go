package subcommand

import (
	"gitlab.com/lightmeter/controlcenter/logdb"
	"gitlab.com/lightmeter/controlcenter/logeater"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/workspace"
	"io"
	"log"
	"time"
)

func OnlyImportLogs(workspaceDirectory string, timezone *time.Location, logYear int, verbose bool, reader io.Reader) {
	ws, err := workspace.NewWorkspace(workspaceDirectory, logdb.Config{
		Location: timezone,
	})

	if err != nil {
		errorutil.Die(verbose, errorutil.Wrap(err), "Error creating / opening workspace directory for storing application files:", workspaceDirectory, ". Try specifying a different directory (using -workspace), or check you have permission to write to the specified location.")
	}

	doneWithDatabase := ws.Run()

	initialLogsTime := logeater.BuildInitialLogsTime(ws.MostRecentLogTime(), logYear, timezone)

	publisher := ws.NewPublisher()

	logeater.ParseLogsFromReader(publisher, initialLogsTime, reader)

	publisher.Close()

	<-doneWithDatabase

	log.Println("Importing has finished. Bye!")
}
