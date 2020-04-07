package api

import (
	"encoding/json"
	"gitlab.com/lightmeter/controlcenter/dashboard"
	"gitlab.com/lightmeter/controlcenter/data"
	"gitlab.com/lightmeter/controlcenter/util"
	"gitlab.com/lightmeter/postfix-log-parser"
	"log"
	"net/http"
	"time"
)

func serveJson(w http.ResponseWriter, r *http.Request, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	encoded, err := json.Marshal(v)
	util.MustSucceed(err, "Encoding as JSON in the http API")
	w.Write(encoded)
}

func requestWithInterval(timezone *time.Location,
	w http.ResponseWriter,
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

func HttpDashboard(mux *http.ServeMux, timezone *time.Location, dashboard dashboard.Dashboard) {
	mux.HandleFunc("/api/countByStatus", func(w http.ResponseWriter, r *http.Request) {
		requestWithInterval(timezone, w, r, func(interval data.TimeInterval) {
			serveJson(w, r, map[string]int{
				"sent":     dashboard.CountByStatus(parser.SentStatus, interval),
				"deferred": dashboard.CountByStatus(parser.DeferredStatus, interval),
				"bounced":  dashboard.CountByStatus(parser.BouncedStatus, interval),
			})
		})
	})

	mux.HandleFunc("/api/topBusiestDomains", func(w http.ResponseWriter, r *http.Request) {
		requestWithInterval(timezone, w, r, func(interval data.TimeInterval) {
			serveJson(w, r, dashboard.TopBusiestDomains(interval))
		})
	})

	mux.HandleFunc("/api/topBouncedDomains", func(w http.ResponseWriter, r *http.Request) {
		requestWithInterval(timezone, w, r, func(interval data.TimeInterval) {
			serveJson(w, r, dashboard.TopBouncedDomains(interval))
		})
	})

	mux.HandleFunc("/api/topDeferredDomains", func(w http.ResponseWriter, r *http.Request) {
		requestWithInterval(timezone, w, r, func(interval data.TimeInterval) {
			serveJson(w, r, dashboard.TopDeferredDomains(interval))
		})
	})

	mux.HandleFunc("/api/deliveryStatus", func(w http.ResponseWriter, r *http.Request) {
		requestWithInterval(timezone, w, r, func(interval data.TimeInterval) {
			serveJson(w, r, dashboard.DeliveryStatus(interval))
		})
	})
}
