package main

import (
	"flag"
	"fmt"
	"gitlab.com/lightmeter/controlcenter/logeater"
	"gitlab.com/lightmeter/controlcenter/logeater/dirwatcher"
	"gitlab.com/lightmeter/controlcenter/server"
	"gitlab.com/lightmeter/controlcenter/subcommand"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/version"
	"log"
	"os"
	"time"

	"gitlab.com/lightmeter/controlcenter/domainmapping"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/logdb"
	"gitlab.com/lightmeter/controlcenter/workspace"
)

func main() {
	var (
		shouldWatchFromStdin      bool
		workspaceDirectory        string
		importOnly                bool
		migrateDownToOnly         bool
		migrateDownToVersion      int
		migrateDownToDatabaseName string
		showVersion               bool
		dirToWatch                string
		address                   string
		verbose                   bool
		frontendv2                bool
		emailToPasswdReset        string
		passwordToReset           string

		timezone *time.Location = time.UTC
		logYear  int
	)

	flag.BoolVar(&shouldWatchFromStdin, "stdin", false, "Read log lines from stdin")
	flag.StringVar(&workspaceDirectory, "workspace", "/var/lib/lightmeter_workspace", "Path to the directory to store all working data")
	flag.BoolVar(&importOnly, "importonly", false,
		"Only import logs from stdin, exiting immediately, without running the full application. Implies -stdin")
	flag.BoolVar(&migrateDownToOnly, "migrate_down_to_only", false,
		"Only migrates down")
	flag.StringVar(&migrateDownToDatabaseName, "migrate_down_to_database", "", "Database name only for migration")
	flag.IntVar(&migrateDownToVersion, "migrate_down_to_version", -1, "Specify the new migration version")
	flag.IntVar(&logYear, "log_starting_year", time.Now().Year(), "Value to be used as initial year when it cannot be obtained fro the Postfix logs. Defaults to the current year. Requires -stdin.")
	flag.BoolVar(&showVersion, "version", false, "Show Version Information")
	flag.StringVar(&dirToWatch, "watch_dir", "", "Path to the directory where postfix stores its log files, to be watched")
	flag.StringVar(&address, "listen", ":8080", "Network address to listen to")
	flag.BoolVar(&verbose, "verbose", false, "Be Verbose")
	flag.StringVar(&emailToPasswdReset, "email_reset", "", "Reset password for user (implies -password and depends on -workspace)")
	flag.StringVar(&passwordToReset, "password", "", "Password to reset (requires -email_reset)")
	flag.BoolVar(&frontendv2, "frontendv2", false, "use frontend v2")

	flag.Usage = func() {
		printVersion()
		fmt.Fprintf(os.Stdout, "\n Example call: \n")
		fmt.Fprintf(os.Stdout, "\n %s -workspace ~/lightmeter_workspace -watch_dir /var/log \n", os.Args[0])
		fmt.Fprintf(os.Stdout, "\n Flag set: \n\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if showVersion {
		printVersion()
		return
	}

	lmsqlite3.Initialize(lmsqlite3.Options{"domain_mapping": domainmapping.DefaultMapping})

	if migrateDownToOnly {
		subcommand.PerformMigrateDownTo(verbose, workspaceDirectory, migrateDownToDatabaseName, int64(migrateDownToVersion))
		return
	}

	if len(emailToPasswdReset) > 0 {
		subcommand.PerformPasswordReset(verbose, workspaceDirectory, emailToPasswdReset, passwordToReset)
		return
	}

	if len(dirToWatch) == 0 && !shouldWatchFromStdin && !importOnly {
		errorutil.Die(verbose, nil, "No logs sources specified or import flag provided! Use -help to more info.")
	}

	if importOnly {
		subcommand.OnlyImportLogs(workspaceDirectory, timezone, logYear, verbose, os.Stdin)
		return
	}

	ws, err := workspace.NewWorkspace(workspaceDirectory, logdb.Config{
		Location: timezone,
	})

	if err != nil {
		errorutil.Die(verbose, errorutil.Wrap(err), "Error creating / opening workspace directory for storing application files:", workspaceDirectory, ". Try specifying a different directory (using -workspace), or check you have permission to write to the specified location.")
	}

	go func() {
		<-ws.Run()
		log.Panicln("Error: Workspace execution has ended, which should never happen here!")
	}()

	func() {
		if len(dirToWatch) > 0 {
			runWatchingDirectory(&ws, dirToWatch, verbose)
			return
		}

		if shouldWatchFromStdin {
			watchFromStdin(&ws, logYear, timezone)
			return
		}
	}()

	httpServer := server.HttpServer{
		Workspace:          &ws,
		WorkspaceDirectory: workspaceDirectory,
		Timezone:           timezone,
		Address:            address,
	}

	if frontendv2 {
		errorutil.MustSucceed(httpServer.StartV2(), "server died")
	} else {
		errorutil.MustSucceed(httpServer.Start(), "server died")
	}
}

func printVersion() {
	fmt.Fprintf(os.Stderr, "Lightmeter ControlCenter %s\n", version.Version)
}

func runWatchingDirectory(ws *workspace.Workspace, dirToWatch string, verbose bool) {
	dir, err := dirwatcher.NewDirectoryContent(dirToWatch)

	if err != nil {
		errorutil.Die(verbose, errorutil.Wrap(err), "Error opening directory:", dirToWatch)
	}

	initialTime := ws.MostRecentLogTime()

	func() {
		if initialTime.IsZero() {
			log.Println("Start importing Postfix logs directory into a new workspace")
			return
		}

		log.Println("Importing Postfix logs directory from time", initialTime)
	}()

	watcher := dirwatcher.NewDirectoryImporter(dir, ws.NewPublisher(), initialTime)

	go func() {
		if err := watcher.Run(); err != nil {
			fmt.Println(err)
			errorutil.Die(verbose, errorutil.Wrap(err), "Error watching directory:", dirToWatch)
		}
	}()
}

func watchFromStdin(ws *workspace.Workspace, logYear int, timezone *time.Location) {
	initialLogsTime := logeater.BuildInitialLogsTime(ws.MostRecentLogTime(), logYear, timezone)
	go logeater.ParseLogsFromReader(ws.NewPublisher(), initialLogsTime, os.Stdin)
}
