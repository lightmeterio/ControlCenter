package main

import (
	"flag"
	"fmt"
	"gitlab.com/lightmeter/controlcenter/domainmapping"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/util/dbutil"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"gitlab.com/lightmeter/controlcenter/api"
	"gitlab.com/lightmeter/controlcenter/auth"
	"gitlab.com/lightmeter/controlcenter/data"
	"gitlab.com/lightmeter/controlcenter/httpauth"
	"gitlab.com/lightmeter/controlcenter/httpsettings"
	"gitlab.com/lightmeter/controlcenter/i18n"
	"gitlab.com/lightmeter/controlcenter/logdb"
	"gitlab.com/lightmeter/controlcenter/logeater"
	"gitlab.com/lightmeter/controlcenter/logeater/dirwatcher"
	"gitlab.com/lightmeter/controlcenter/po"
	"gitlab.com/lightmeter/controlcenter/staticdata"
	"gitlab.com/lightmeter/controlcenter/version"
	"gitlab.com/lightmeter/controlcenter/workspace"
)

type watchableFilenames []string

func (this watchableFilenames) String() string {
	return strings.Join(this, ", ")
}

func (this *watchableFilenames) Set(value string) error {
	*this = append(*this, value)
	return nil
}

var (
	filesToWatch              watchableFilenames
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
	emailToPasswdReset        string
	passwordToReset           string

	timezone *time.Location = time.UTC
	logYear  int
)

func printVersion() {
	fmt.Fprintf(os.Stderr, "Lightmeter ControlCenter %s\n", version.Version)
}

func init() {
	flag.Var(&filesToWatch, "watch_file", "File to watch (can be used multiple times)")
	flag.BoolVar(&shouldWatchFromStdin, "stdin", false, "Read log lines from stdin")
	flag.StringVar(&workspaceDirectory, "workspace", "/var/lib/lightmeter_workspace", "Path to the directory to store all working data")
	flag.BoolVar(&importOnly, "importonly", false,
		"Only import logs from stdin, exiting immediately, without running the full application. Implies -stdin")
	flag.BoolVar(&migrateDownToOnly, "migrate_down_to_only", false,
		"Only migrates down")
	flag.StringVar(&migrateDownToDatabaseName, "migrate_down_to_database", "", "Database name only for migration")
	flag.IntVar(&migrateDownToVersion, "migrate_down_to_version", -1, "Specify the new migration version")
	flag.IntVar(&logYear, "what_year_is_it", time.Now().Year(), "Specify the year when the logs start. Defaults to the current year. This option is temporary and will be removed soon. Promise :-)")
	flag.BoolVar(&showVersion, "version", false, "Show Version Information")
	flag.StringVar(&dirToWatch, "watch_dir", "", "Path to the directory where postfix stores its log files, to be watched")
	flag.StringVar(&address, "listen", ":8080", "Network address to listen to")
	flag.BoolVar(&verbose, "verbose", false, "Be Verbose")
	flag.StringVar(&emailToPasswdReset, "email_reset", "", "Reset password for user (implies -password and depends on -workspace)")
	flag.StringVar(&passwordToReset, "password", "", "Password to reset (requires -email_reset)")

	flag.Usage = func() {
		printVersion()
		fmt.Fprintf(os.Stdout, "\n Example call: \n")
		fmt.Fprintf(os.Stdout, "\n %s -workspace ~/lightmeter_workspace -watch_dir /var/log \n", os.Args[0])
		fmt.Fprintf(os.Stdout, "\n Flag set: \n\n")
		flag.PrintDefaults()
	}
}

func performPasswordReset() {
	auth, err := auth.NewAuth(workspaceDirectory, auth.Options{})

	if err != nil {
		errorutil.Die(verbose, errorutil.Wrap(err), "Error opening auth database:", err)
	}

	if err := auth.ChangePassword(emailToPasswdReset, passwordToReset); err != nil {
		errorutil.Die(verbose, errorutil.Wrap(err), "Error resetting password:", err)
	}

	if err := auth.Close(); err != nil {
		errorutil.Die(verbose, errorutil.Wrap(err), "Error closing auth database:", err)
	}

	log.Println("Password for user", emailToPasswdReset, "reset successfully")
}

// The downgrade of multiple database is disallowed to prevent to many failures
func performMigrateDownTo(workspaceDirectory, databaseName string, version int64) {
	if workspaceDirectory == "" {
		errorutil.Die(verbose, nil, "No workspace dir specified! Use -help to more info.")
	}
	if migrateDownToDatabaseName == "" {
		errorutil.Die(verbose, nil, "No database name specified! Use -help to more info.")
	}
	if version == -1 {
		errorutil.Die(verbose, nil, "No migration version specified! Use -help to more info.")
	}

	err := dbutil.MigratorRunDown(workspaceDirectory, databaseName, version)
	if err != nil {
		errorutil.Die(verbose, errorutil.Wrap(err), fmt.Sprintf("Error %s migrate down to version", databaseName))
	}
	log.Println("migrated database down to version for ", databaseName, " successfully")
}

func runWatchingDirectory(ws *workspace.Workspace) {
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

func runWatchingFiles(ws *workspace.Workspace) {
	logFilesWatchLocation := logeater.FindWatchingLocationForWorkspace(ws)

	for _, filename := range filesToWatch {
		log.Println("Now watching file", filename, "for changes from the", func() string {
			if logFilesWatchLocation.Whence == os.SEEK_END {
				return "end"
			}

			return "beginning"
		}())

		go func(filename string) {
			if err := logeater.WatchFile(filename, logFilesWatchLocation, ws.NewPublisher(), buildInitialLogsTime(ws)); err != nil {
				errorutil.Die(verbose, errorutil.Wrap(err), "Error watching file:", filename)
			}
		}(filename)
	}
}

func watchFromStdin(ws *workspace.Workspace) {
	go parseLogsFromStdin(ws.NewPublisher(), buildInitialLogsTime(ws))
}

func buildInitialLogsTime(ws *workspace.Workspace) time.Time {
	ts := ws.MostRecentLogTime()

	if !ts.IsZero() {
		return ts
	}

	return time.Date(logYear, time.January, 1, 0, 0, 0, 0, timezone)
}

func startHTTPServer(ws *workspace.Workspace) {
	settings := ws.Settings()

	initialSetupHandler := httpsettings.NewInitialSetupHandler(settings)

	mux := http.NewServeMux()

	mux.Handle("/", i18n.DefaultWrap(http.FileServer(staticdata.HttpAssets), staticdata.HttpAssets, po.DefaultCatalog))

	exposeApiExplorer(mux)

	exposeProfiler(mux)

	dashboard, err := ws.Dashboard()

	if err != nil {
		errorutil.Die(verbose, errorutil.Wrap(err), "Error creating dashboard")
	}

	insightsFetcher := ws.InsightsFetcher()

	api.HttpDashboard(mux, timezone, dashboard)

	api.HttpInsights(mux, timezone, insightsFetcher)

	mux.Handle("/settings/initialSetup", initialSetupHandler)

	// Some paths that don't require authentication
	// That's what people nowadays call a "allow list".
	publicPaths := []string{
		"/img",
		"/css",
		"/fonts",
		"/js",
		"/3rd",
		"/debug",
	}

	authWrapper := httpauth.Serve(mux, ws.Auth(), workspaceDirectory, publicPaths)

	log.Fatal(http.ListenAndServe(address, authWrapper))
}

func main() {
	flag.Parse()

	if showVersion {
		printVersion()
		return
	}

	lmsqlite3.Initialize(lmsqlite3.Options{"domain_mapping": domainmapping.DefaultMapping})

	if migrateDownToOnly {
		performMigrateDownTo(workspaceDirectory, migrateDownToDatabaseName, int64(migrateDownToVersion))
		return
	}

	if len(emailToPasswdReset) > 0 {
		performPasswordReset()
		return
	}

	if len(dirToWatch) == 0 && !shouldWatchFromStdin && len(filesToWatch) == 0 && !importOnly {
		errorutil.Die(verbose, nil, "No logs sources specified or import flag provided! Use -help to more info.")
	}

	ws, err := workspace.NewWorkspace(workspaceDirectory, logdb.Config{
		Location: timezone,
	})

	if err != nil {
		errorutil.Die(verbose, errorutil.Wrap(err), "Error creating / opening workspace directory for storing application files:", workspaceDirectory, ". Try specifying a different directory (using -workspace), or check you have permission to write to the specified location.")
	}

	doneWithDatabase := ws.Run()

	if importOnly {
		parseLogsFromStdin(ws.NewPublisher(), buildInitialLogsTime(&ws))
		<-doneWithDatabase
		log.Println("Importing has finished. Bye!")
		return
	}

	func() {
		if len(dirToWatch) > 0 {
			runWatchingDirectory(&ws)
			return
		}

		if shouldWatchFromStdin {
			watchFromStdin(&ws)
			return
		}

		if len(filesToWatch) > 0 {
			runWatchingFiles(&ws)
			return
		}
	}()

	startHTTPServer(&ws)
}

func parseLogsFromStdin(publisher data.Publisher, ts time.Time) {
	logeater.ReadFromReader(os.Stdin, publisher, ts)
	publisher.Close()
	log.Println("STDIN has just closed!")
}
