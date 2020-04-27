package api

import (
	"encoding/json"
	"net/http"
	"time"

	"gitlab.com/lightmeter/controlcenter/dashboard"
	"gitlab.com/lightmeter/controlcenter/data"
	"gitlab.com/lightmeter/controlcenter/util"
	parser "gitlab.com/lightmeter/postfix-log-parser"
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
		http.Error(w, "Wrong input", http.StatusUnprocessableEntity)
		return
	}

	interval, err := data.ParseTimeInterval(r.Form.Get("from"), r.Form.Get("to"), timezone)

	if err != nil {
		http.Error(w, "Error parsing time interval:\""+err.Error()+"\"", http.StatusUnprocessableEntity)
		return
	}

	onParserSuccess(interval)
}

type handler struct {
	dashboard dashboard.Dashboard
	timezone  *time.Location
}

type countByStatusHandler handler

func (h countByStatusHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requestWithInterval(h.timezone, w, r, func(interval data.TimeInterval) {
		serveJson(w, r, map[string]int{
			"sent":     h.dashboard.CountByStatus(parser.SentStatus, interval),
			"deferred": h.dashboard.CountByStatus(parser.DeferredStatus, interval),
			"bounced":  h.dashboard.CountByStatus(parser.BouncedStatus, interval),
		})
	})
}

type topBusiestDomainsHandler handler

func (h topBusiestDomainsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requestWithInterval(h.timezone, w, r, func(interval data.TimeInterval) {
		serveJson(w, r, h.dashboard.TopBusiestDomains(interval))
	})
}

type topBouncedDomainsHandler handler

func (h topBouncedDomainsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requestWithInterval(h.timezone, w, r, func(interval data.TimeInterval) {
		serveJson(w, r, h.dashboard.TopBouncedDomains(interval))
	})
}

type topDeferredDomainsHandler handler

func (h topDeferredDomainsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requestWithInterval(h.timezone, w, r, func(interval data.TimeInterval) {
		serveJson(w, r, h.dashboard.TopDeferredDomains(interval))
	})
}

type deliveryStatusHandler handler

func (h deliveryStatusHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requestWithInterval(h.timezone, w, r, func(interval data.TimeInterval) {
		serveJson(w, r, h.dashboard.DeliveryStatus(interval))
	})
}

func HttpDashboard(mux *http.ServeMux, timezone *time.Location, dashboard dashboard.Dashboard) {
	mux.Handle("/api/countByStatus", countByStatusHandler{dashboard, timezone})
	mux.Handle("/api/topBusiestDomains", topBusiestDomainsHandler{dashboard, timezone})
	mux.Handle("/api/topBouncedDomains", topBouncedDomainsHandler{dashboard, timezone})
	mux.Handle("/api/topDeferredDomains", topDeferredDomainsHandler{dashboard, timezone})
	mux.Handle("/api/deliveryStatus", deliveryStatusHandler{dashboard, timezone})
}
