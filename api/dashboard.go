package api

import (
	"net/http"
	"time"

	"gitlab.com/lightmeter/controlcenter/dashboard"
	"gitlab.com/lightmeter/controlcenter/data"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/version"

	parser "gitlab.com/lightmeter/postfix-log-parser"
)

type handler struct {
	//nolint:structcheck
	dashboard dashboard.Dashboard
	//nolint:structcheck
	timezone *time.Location
}

type countByStatusHandler handler

type countByStatusResult map[string]int

// @Summary Count By Status
// @Param from query string true "Initial date in the format 1999-12-23"
// @Param to   query string true "Final date in the format 1999-12-23"
// @Produce json
// @Success 200 {object} countByStatusResult "desc"
// @Failure 422 {string} string "desc"
// @Router /api/v0/countByStatus [get]
func (h countByStatusHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requestWithInterval(h.timezone, w, r, func(interval data.TimeInterval) {
		sent, err := h.dashboard.CountByStatus(parser.SentStatus, interval)
		errorutil.MustSucceed(err, "")

		deferred, err := h.dashboard.CountByStatus(parser.DeferredStatus, interval)
		errorutil.MustSucceed(err, "")

		bounced, err := h.dashboard.CountByStatus(parser.BouncedStatus, interval)
		errorutil.MustSucceed(err, "")

		serveJson(w, r, countByStatusResult{
			"sent":     sent,
			"deferred": deferred,
			"bounced":  bounced,
		})
	})
}

func servePairsFromTimeInterval(w http.ResponseWriter, r *http.Request, f func(data.TimeInterval) (dashboard.Pairs, error), interval data.TimeInterval) {
	pairs, err := f(interval)
	errorutil.MustSucceed(err, "")
	serveJson(w, r, pairs)
}

type topBusiestDomainsHandler handler

// @Summary Top Busiest Domains
// @Param from query string true "Initial date in the format 1999-12-23"
// @Param to   query string true "Final date in the format 1999-12-23"
// @Produce json
// @Success 200 {object} dashboard.Pairs
// @Failure 422 {string} string "desc"
// @Router /api/v0/topBusiestDomains [get]
func (h topBusiestDomainsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requestWithInterval(h.timezone, w, r, func(interval data.TimeInterval) {
		servePairsFromTimeInterval(w, r, h.dashboard.TopBusiestDomains, interval)
	})
}

type topBouncedDomainsHandler handler

// @Summary Top Bounced Domains
// @Param from query string true "Initial date in the format 1999-12-23"
// @Param to   query string true "Final date in the format 1999-12-23"
// @Produce json
// @Success 200 {object} dashboard.Pairs
// @Failure 422 {string} string "desc"
// @Router /api/v0/topBouncedDomains [get]
func (h topBouncedDomainsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requestWithInterval(h.timezone, w, r, func(interval data.TimeInterval) {
		servePairsFromTimeInterval(w, r, h.dashboard.TopBouncedDomains, interval)
	})
}

type topDeferredDomainsHandler handler

// @Summary Top Deferred Domains
// @Param from query string true "Initial date in the format 1999-12-23"
// @Param to   query string true "Final date in the format 1999-12-23"
// @Produce json
// @Success 200 {object} dashboard.Pairs
// @Failure 422 {string} string "desc"
// @Router /api/v0/topDeferredDomains [get]
func (h topDeferredDomainsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requestWithInterval(h.timezone, w, r, func(interval data.TimeInterval) {
		servePairsFromTimeInterval(w, r, h.dashboard.TopDeferredDomains, interval)
	})
}

type deliveryStatusHandler handler

// @Summary Delivery Status
// @Param from query string true "Initial date in the format 1999-12-23"
// @Param to   query string true "Final date in the format 1999-12-23"
// @Produce json
// @Success 200 {object} dashboard.Pairs
// @Failure 422 {string} string "desc"
// @Router /api/v0/deliveryStatus [get]
func (h deliveryStatusHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requestWithInterval(h.timezone, w, r, func(interval data.TimeInterval) {
		servePairsFromTimeInterval(w, r, h.dashboard.DeliveryStatus, interval)
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
// @Router /api/v0/appVersion [get]
func (appVersionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	serveJson(w, r, appVersion{Version: version.Version, Commit: version.Commit, TagOrBranch: version.TagOrBranch})
}

func HttpDashboard(mux *http.ServeMux, timezone *time.Location, dashboard dashboard.Dashboard) {
	mux.Handle("/api/v0/countByStatus", countByStatusHandler{dashboard, timezone})
	mux.Handle("/api/v0/topBusiestDomains", topBusiestDomainsHandler{dashboard, timezone})
	mux.Handle("/api/v0/topBouncedDomains", topBouncedDomainsHandler{dashboard, timezone})
	mux.Handle("/api/v0/topDeferredDomains", topDeferredDomainsHandler{dashboard, timezone})
	mux.Handle("/api/v0/deliveryStatus", deliveryStatusHandler{dashboard, timezone})
	mux.Handle("/api/v0/appVersion", appVersionHandler{})
}
