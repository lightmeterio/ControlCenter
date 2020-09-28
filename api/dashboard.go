package api

import (
	"log"
	"net/http"
	"time"

	"gitlab.com/lightmeter/controlcenter/dashboard"
	"gitlab.com/lightmeter/controlcenter/data"
	"gitlab.com/lightmeter/controlcenter/httpmiddleware"
	"gitlab.com/lightmeter/controlcenter/version"

	parser "gitlab.com/lightmeter/postfix-log-parser"
)

type handler struct {
	//nolint:structcheck
	dashboard dashboard.Dashboard
}

type countByStatusHandler handler

type countByStatusResult map[string]int

func serveError(w http.ResponseWriter, r *http.Request, err error) {
	log.Println(err)
	w.WriteHeader(http.StatusInternalServerError)
}

// @Summary Count By Status
// @Param from query string true "Initial date in the format 1999-12-23"
// @Param to   query string true "Final date in the format 1999-12-23"
// @Produce json
// @Success 200 {object} countByStatusResult "desc"
// @Failure 422 {string} string "desc"
// @Router /api/v0/countByStatus [get]
func (h countByStatusHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	interval := httpmiddleware.GetIntervalFromContext(r)

	sent, err := h.dashboard.CountByStatus(parser.SentStatus, interval)
	if err != nil {
		serveError(w, r, err)
		return
	}

	deferred, err := h.dashboard.CountByStatus(parser.DeferredStatus, interval)
	if err != nil {
		serveError(w, r, err)
		return
	}

	bounced, err := h.dashboard.CountByStatus(parser.BouncedStatus, interval)
	if err != nil {
		serveError(w, r, err)
		return
	}

	serveJson(w, r, countByStatusResult{
		"sent":     sent,
		"deferred": deferred,
		"bounced":  bounced,
	})
}

func servePairsFromTimeInterval(
	w http.ResponseWriter,
	r *http.Request,
	f func(data.TimeInterval) (dashboard.Pairs, error),
	interval data.TimeInterval) {
	pairs, err := f(interval)
	if err != nil {
		serveError(w, r, err)
		return
	}

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
	interval := httpmiddleware.GetIntervalFromContext(r)
	servePairsFromTimeInterval(w, r, h.dashboard.TopBusiestDomains, interval)
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
	interval := httpmiddleware.GetIntervalFromContext(r)
	servePairsFromTimeInterval(w, r, h.dashboard.TopBouncedDomains, interval)
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
	interval := httpmiddleware.GetIntervalFromContext(r)
	servePairsFromTimeInterval(w, r, h.dashboard.TopDeferredDomains, interval)
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
	interval := httpmiddleware.GetIntervalFromContext(r)
	servePairsFromTimeInterval(w, r, h.dashboard.DeliveryStatus, interval)
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
	mux.Handle("/api/v0/countByStatus", httpmiddleware.RequestWithInterval(timezone)(countByStatusHandler{dashboard}))
	mux.Handle("/api/v0/topBusiestDomains", httpmiddleware.RequestWithInterval(timezone)(topBusiestDomainsHandler{dashboard}))
	mux.Handle("/api/v0/topBouncedDomains", httpmiddleware.RequestWithInterval(timezone)(topBouncedDomainsHandler{dashboard}))
	mux.Handle("/api/v0/topDeferredDomains", httpmiddleware.RequestWithInterval(timezone)(topDeferredDomainsHandler{dashboard}))
	mux.Handle("/api/v0/deliveryStatus", httpmiddleware.RequestWithInterval(timezone)(deliveryStatusHandler{dashboard}))
	mux.Handle("/api/v0/appVersion", appVersionHandler{})
}
