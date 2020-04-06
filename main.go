package main

import (
	"encoding/json"
	"flag"
	"gitlab.com/lightmeter/controlcenter/data"
	"gitlab.com/lightmeter/controlcenter/logeater"
	"gitlab.com/lightmeter/controlcenter/staticdata"
	"gitlab.com/lightmeter/controlcenter/workspace"
	parser "gitlab.com/lightmeter/postfix-log-parser"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
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

	timezone *time.Location = time.UTC
	logYear  int
)

func init() {
	flag.Var(&filesToWatch, "watch", "File to watch (can be used multiple times)")
	flag.BoolVar(&watchFromStdin, "stdin", false, "Read log lines from stdin")
	flag.StringVar(&workspaceDirectory, "workspace", "lightmeter_workspace", "Path to the directory to store all working data")
	flag.BoolVar(&importOnly, "importonly", false,
		"Only import logs from stdin, exiting imediately, without running the full application. Implies -stdin")
	flag.IntVar(&logYear, "what_year_is_it", time.Now().Year(), "Specify the year when the logs start. Defaults to the current year. This option is temporary and will be removed soon. Promise :-)")
}

func main() {
	flag.Parse()

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

	serveJson := func(w http.ResponseWriter, r *http.Request, v interface{}) {
		w.Header().Set("Content-Type", "application/json")
		encoded, _ := json.Marshal(v)
		w.Write(encoded)
	}

	requestWithInterval := func(w http.ResponseWriter,
		r *http.Request,
		onParserSuccess func(interval data.TimeInterval)) {

		if r.ParseForm() != nil {
			log.Println("Error parsing form!")
			serveJson(w, r, []int{})
			return
		}

		interval, err := data.ParseTimeInterval(r.Form.Get("from"), r.Form.Get("to"), timezone)

		if err != nil {
			log.Println("Error parsing time interval:", err)
			serveJson(w, r, []int{})
			return
		}

		onParserSuccess(interval)
	}

	dashboard, err := ws.Dashboard()

	if err != nil {
		log.Fatal("Error building dashboard:", err)
	}

	http.HandleFunc("/api/countByStatus", func(w http.ResponseWriter, r *http.Request) {
		requestWithInterval(w, r, func(interval data.TimeInterval) {
			serveJson(w, r, map[string]int{
				"sent":     dashboard.CountByStatus(parser.SentStatus, interval),
				"deferred": dashboard.CountByStatus(parser.DeferredStatus, interval),
				"bounced":  dashboard.CountByStatus(parser.BouncedStatus, interval),
			})
		})
	})

	http.HandleFunc("/api/topBusiestDomains", func(w http.ResponseWriter, r *http.Request) {
		requestWithInterval(w, r, func(interval data.TimeInterval) {
			serveJson(w, r, dashboard.TopBusiestDomains(interval))
		})
	})

	http.HandleFunc("/api/topBouncedDomains", func(w http.ResponseWriter, r *http.Request) {
		requestWithInterval(w, r, func(interval data.TimeInterval) {
			serveJson(w, r, dashboard.TopBouncedDomains(interval))
		})
	})

	http.HandleFunc("/api/topDeferredDomains", func(w http.ResponseWriter, r *http.Request) {
		requestWithInterval(w, r, func(interval data.TimeInterval) {
			serveJson(w, r, dashboard.TopDeferredDomains(interval))
		})
	})

	http.HandleFunc("/api/deliveryStatus", func(w http.ResponseWriter, r *http.Request) {
		requestWithInterval(w, r, func(interval data.TimeInterval) {
			serveJson(w, r, dashboard.DeliveryStatus(interval))
		})
	})

	http.Handle("/", http.FileServer(staticdata.HttpAssets))

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func parseLogsFromStdin(publisher data.Publisher) {
	logeater.ReadFromReader(os.Stdin, publisher)
	publisher.Close()
	log.Println("STDIN has just closed!")
}
