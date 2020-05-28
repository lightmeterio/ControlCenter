package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"gitlab.com/lightmeter/controlcenter/logeater/dirwatcher"
	"gitlab.com/lightmeter/controlcenter/util"

	"gitlab.com/lightmeter/controlcenter/api"
	"gitlab.com/lightmeter/controlcenter/data"
	"gitlab.com/lightmeter/controlcenter/logeater"
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
	filesToWatch       watchableFilenames
	watchFromStdin     bool
	workspaceDirectory string
	importOnly         bool
	showVersion        bool
	dirToWatch         string

	timezone *time.Location = time.UTC
	logYear  int
)

func printVersion() {
	fmt.Fprintf(os.Stderr, "Lightmeter ControlCenter %s\n", version.Version)
}

func init() {
	flag.Var(&filesToWatch, "watch", "File to watch (can be used multiple times)")
	flag.BoolVar(&watchFromStdin, "stdin", false, "Read log lines from stdin")
	flag.StringVar(&workspaceDirectory, "workspace", "lightmeter_workspace", "Path to the directory to store all working data")
	flag.BoolVar(&importOnly, "importonly", false,
		"Only import logs from stdin, exiting immediately, without running the full application. Implies -stdin")
	flag.IntVar(&logYear, "what_year_is_it", time.Now().Year(), "Specify the year when the logs start. Defaults to the current year. This option is temporary and will be removed soon. Promise :-)")
	flag.BoolVar(&showVersion, "version", false, "Show Version Information")
	flag.StringVar(&dirToWatch, "watch_dir", "", "Path to the directory where postfix stores its log files, to be watched")

	flag.Usage = func() {
		printVersion()
		flag.PrintDefaults()
	}
}

func main() {
	flag.Parse()

	if showVersion {
		printVersion()
		return
	}

	postfixLogsDirContent := func() dirwatcher.DirectoryContent {
		if len(dirToWatch) != 0 {
			dir, err := dirwatcher.NewDirectoryContent(dirToWatch)
			util.MustSucceed(err, "Opening directory: "+dirToWatch)
			return dir
		}

		return nil
	}()

	if postfixLogsDirContent != nil {
		initialLogTimeFromDirectory, err := dirwatcher.FindInitialLogTime(postfixLogsDirContent)
		util.MustSucceed(err, "Obtaining initial time from directory: "+dirToWatch)
		log.Println("Using initial time from postfix log directory:", initialLogTimeFromDirectory)
		logYear = initialLogTimeFromDirectory.Year()
	}

	ws, err := workspace.NewWorkspace(workspaceDirectory, data.Config{
		Location:    timezone,
		DefaultYear: logYear,
	})

	if err != nil {
		log.Fatal("Could not initialize workspace:", err)
	}

	doneWithDatabase := ws.Run()

	pub := ws.NewPublisher()

	if importOnly {
		parseLogsFromStdin(pub)
		<-doneWithDatabase
		log.Println("Importing has finished. Bye!")
		return
	}

	if watchFromStdin {
		go parseLogsFromStdin(pub)
	}

	logFilesWatchLocation := logeater.FindWatchingLocationForWorkspace(&ws)

	for _, filename := range filesToWatch {
		log.Println("Now watching file", filename, "for changes from the", func() string {
			if logFilesWatchLocation.Whence == os.SEEK_END {
				return "end"
			}

			return "beginning"
		}())

		go func(filename string) {
			if err := logeater.WatchFile(filename, logFilesWatchLocation, pub); err != nil {
				log.Fatal("Failed watching file: ", filename, ", error: ", err)
			}
		}(filename)
	}

	if postfixLogsDirContent != nil {
		initialTime := func() time.Time {
			t := ws.MostRecentLogTime()

			if t.IsZero() {
				return time.Date(1970, time.January, 1, 0, 0, 0, 0, timezone)
			}

			return t
		}()

		log.Println("Start importing Postfix logs directory from time", initialTime)

		watcher := dirwatcher.NewDirectoryImporter(postfixLogsDirContent, pub, timezone, initialTime)

		go func() {
			util.MustSucceed(watcher.Run(), "Watching directory")
		}()
	}

	dashboard, err := ws.Dashboard()

	if err != nil {
		log.Fatal("Error building dashboard:", err)
	}

	mux := http.NewServeMux()

	exposeApiExplorer(mux)

	api.HttpDashboard(mux, timezone, dashboard)

	mux.Handle("/", http.FileServer(staticdata.HttpAssets))

	log.Fatal(http.ListenAndServe(":8080", mux))
}

func parseLogsFromStdin(publisher data.Publisher) {
	logeater.ReadFromReader(os.Stdin, publisher)
	publisher.Close()
	log.Println("STDIN has just closed!")
}
