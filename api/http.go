package api

import (
	"encoding/json"
	"net/http"
	"time"

	"gitlab.com/lightmeter/controlcenter/dashboard"
	"gitlab.com/lightmeter/controlcenter/data"
	"gitlab.com/lightmeter/controlcenter/util"
	"gitlab.com/lightmeter/controlcenter/version"

	parser "gitlab.com/lightmeter/postfix-log-parser"
)

// @title Lightmeter ControlCenter HTTP API
// @version 0.1
// @description API for user interfaces
// @contact.name Lightmeter Team
// @contact.url http://lightmeter.io
// @contact.email dev@lightmeter.io
// @license.name GNU Affero General Public License 3.0
// @license.url https://www.gnu.org/licenses/agpl-3.0.en.html

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

type countByStatusResult map[string]int

// @Summary Count By Status
// @Param from query string true "Initial date in the format 1999-12-23"
// @Param to   query string true "Final date in the format 1999-12-23"
// @Produce json
// @Success 200 {object} countByStatusResult "desc"
// @Failure 422 {string} string "desc"
// @Router /api/countByStatus [get]
func (h countByStatusHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requestWithInterval(h.timezone, w, r, func(interval data.TimeInterval) {
		serveJson(w, r, countByStatusResult{
			"sent":     h.dashboard.CountByStatus(parser.SentStatus, interval),
			"deferred": h.dashboard.CountByStatus(parser.DeferredStatus, interval),
			"bounced":  h.dashboard.CountByStatus(parser.BouncedStatus, interval),
		})
	})
}

type topBusiestDomainsHandler handler

// @Summary Top Busiest Domains
// @Param from query string true "Initial date in the format 1999-12-23"
// @Param to   query string true "Final date in the format 1999-12-23"
// @Produce json
// @Success 200 {object} dashboard.Pairs
// @Failure 422 {string} string "desc"
// @Router /api/topBusiestDomains [get]
func (h topBusiestDomainsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requestWithInterval(h.timezone, w, r, func(interval data.TimeInterval) {
		serveJson(w, r, h.dashboard.TopBusiestDomains(interval))
	})
}

type topBouncedDomainsHandler handler

// @Summary Top Bounced Domains
// @Param from query string true "Initial date in the format 1999-12-23"
// @Param to   query string true "Final date in the format 1999-12-23"
// @Produce json
// @Success 200 {object} dashboard.Pairs
// @Failure 422 {string} string "desc"
// @Router /api/topBouncedDomains [get]
func (h topBouncedDomainsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requestWithInterval(h.timezone, w, r, func(interval data.TimeInterval) {
		serveJson(w, r, h.dashboard.TopBouncedDomains(interval))
	})
}

type topDeferredDomainsHandler handler

// @Summary Top Deferred Domains
// @Param from query string true "Initial date in the format 1999-12-23"
// @Param to   query string true "Final date in the format 1999-12-23"
// @Produce json
// @Success 200 {object} dashboard.Pairs
// @Failure 422 {string} string "desc"
// @Router /api/topDeferredDomains [get]
func (h topDeferredDomainsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requestWithInterval(h.timezone, w, r, func(interval data.TimeInterval) {
		serveJson(w, r, h.dashboard.TopDeferredDomains(interval))
	})
}

type deliveryStatusHandler handler

// @Summary Delivery Status
// @Param from query string true "Initial date in the format 1999-12-23"
// @Param to   query string true "Final date in the format 1999-12-23"
// @Produce json
// @Success 200 {object} dashboard.Pairs
// @Failure 422 {string} string "desc"
// @Router /api/deliveryStatus [get]
func (h deliveryStatusHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requestWithInterval(h.timezone, w, r, func(interval data.TimeInterval) {
		serveJson(w, r, h.dashboard.DeliveryStatus(interval))
	})
}

type appVersionHandler struct{}

type appVersion struct {
	Version     string
	Commit      string
	TagOrBranch string
}

// @Summary Control Center Version
// @Produce json
// @Success 200 {object} appVersion
// @Router /api/appVersion [get]
func (appVersionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	serveJson(w, r, appVersion{Version: version.Version, Commit: version.Commit, TagOrBranch: version.TagOrBranch})
}

func HttpDashboard(mux *http.ServeMux, timezone *time.Location, dashboard dashboard.Dashboard) {
	mux.Handle("/api/countByStatus", countByStatusHandler{dashboard, timezone})
	mux.Handle("/api/topBusiestDomains", topBusiestDomainsHandler{dashboard, timezone})
	mux.Handle("/api/topBouncedDomains", topBouncedDomainsHandler{dashboard, timezone})
	mux.Handle("/api/topDeferredDomains", topDeferredDomainsHandler{dashboard, timezone})
	mux.Handle("/api/deliveryStatus", deliveryStatusHandler{dashboard, timezone})
	mux.Handle("/api/appVersion", appVersionHandler{})
}
