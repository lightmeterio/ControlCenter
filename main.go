package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

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

		go func() {
			if err := logeater.WatchFile(filename, logFilesWatchLocation, pub); err != nil {
				log.Println("Failed watching file:", filename, "error:", err)
			}
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
