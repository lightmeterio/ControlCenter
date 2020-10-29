package api

import (
	"context"
	"gitlab.com/lightmeter/controlcenter/dashboard"
	"gitlab.com/lightmeter/controlcenter/data"
	"gitlab.com/lightmeter/controlcenter/httpmiddleware"
	parser "gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser"
	"gitlab.com/lightmeter/controlcenter/util/httputil"
	"gitlab.com/lightmeter/controlcenter/version"
	"net/http"
	"time"
)

type handler struct {
	//nolint:structcheck
	dashboard dashboard.Dashboard
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
func (h countByStatusHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) error {
	interval := httpmiddleware.GetIntervalFromContext(r)

	sent, err := h.dashboard.CountByStatus(r.Context(), parser.SentStatus, interval)
	if err != nil {
		return httpmiddleware.NewHTTPStatusCodeError(http.StatusInternalServerError, err)
	}

	deferred, err := h.dashboard.CountByStatus(r.Context(), parser.DeferredStatus, interval)
	if err != nil {
		return httpmiddleware.NewHTTPStatusCodeError(http.StatusInternalServerError, err)
	}

	bounced, err := h.dashboard.CountByStatus(r.Context(), parser.BouncedStatus, interval)
	if err != nil {
		return httpmiddleware.NewHTTPStatusCodeError(http.StatusInternalServerError, err)
	}

	return httputil.WriteJson(w, countByStatusResult{
		"sent":     sent,
		"deferred": deferred,
		"bounced":  bounced,
	}, http.StatusOK)
}

func servePairsFromTimeInterval(
	w http.ResponseWriter,
	r *http.Request,
	f func(context.Context, data.TimeInterval) (dashboard.Pairs, error),
	interval data.TimeInterval) error {
	pairs, err := f(r.Context(), interval)
	if err != nil {
		return err
	}

	return httputil.WriteJson(w, pairs, http.StatusOK)
}

type topBusiestDomainsHandler handler

// @Summary Top Busiest Domains
// @Param from query string true "Initial date in the format 1999-12-23"
// @Param to   query string true "Final date in the format 1999-12-23"
// @Produce json
// @Success 200 {object} dashboard.Pairs
// @Failure 422 {string} string "desc"
// @Router /api/v0/topBusiestDomains [get]
func (h topBusiestDomainsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) error {
	interval := httpmiddleware.GetIntervalFromContext(r)
	return servePairsFromTimeInterval(w, r, h.dashboard.TopBusiestDomains, interval)
}

type topBouncedDomainsHandler handler

// @Summary Top Bounced Domains
// @Param from query string true "Initial date in the format 1999-12-23"
// @Param to   query string true "Final date in the format 1999-12-23"
// @Produce json
// @Success 200 {object} dashboard.Pairs
// @Failure 422 {string} string "desc"
// @Router /api/v0/topBouncedDomains [get]
func (h topBouncedDomainsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) error {
	interval := httpmiddleware.GetIntervalFromContext(r)
	return servePairsFromTimeInterval(w, r, h.dashboard.TopBouncedDomains, interval)
}

type topDeferredDomainsHandler handler

// @Summary Top Deferred Domains
// @Param from query string true "Initial date in the format 1999-12-23"
// @Param to   query string true "Final date in the format 1999-12-23"
// @Produce json
// @Success 200 {object} dashboard.Pairs
// @Failure 422 {string} string "desc"
// @Router /api/v0/topDeferredDomains [get]
func (h topDeferredDomainsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) error {
	interval := httpmiddleware.GetIntervalFromContext(r)
	return servePairsFromTimeInterval(w, r, h.dashboard.TopDeferredDomains, interval)
}

type deliveryStatusHandler handler

// @Summary Delivery Status
// @Param from query string true "Initial date in the format 1999-12-23"
// @Param to   query string true "Final date in the format 1999-12-23"
// @Produce json
// @Success 200 {object} dashboard.Pairs
// @Failure 422 {string} string "desc"
// @Router /api/v0/deliveryStatus [get]
func (h deliveryStatusHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) error {
	interval := httpmiddleware.GetIntervalFromContext(r)
	return servePairsFromTimeInterval(w, r, h.dashboard.DeliveryStatus, interval)
}

type appVersionHandler struct{}

type appVersion struct {
	Version     string `json:"version"`
	Commit      string `json:"commit"`
	TagOrBranch string `json:"tag_or_branch"`
}

// @Summary Control Center Version
// @Produce json
// @Success 200 {object} appVersion
// @Router /api/v0/appVersion [get]
func (appVersionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) error {
	return httputil.WriteJson(w, appVersion{Version: version.Version, Commit: version.Commit, TagOrBranch: version.TagOrBranch}, http.StatusOK)
}

func HttpDashboard(mux *http.ServeMux, timezone *time.Location, dashboard dashboard.Dashboard) {
	chain := httpmiddleware.WithDefaultStack(httpmiddleware.RequestWithInterval(timezone))
	mux.Handle("/api/v0/countByStatus", chain.WithEndpoint(countByStatusHandler{dashboard}))
	mux.Handle("/api/v0/topBusiestDomains", chain.WithEndpoint(topBusiestDomainsHandler{dashboard}))
	mux.Handle("/api/v0/topBouncedDomains", chain.WithEndpoint(topBouncedDomainsHandler{dashboard}))
	mux.Handle("/api/v0/topDeferredDomains", chain.WithEndpoint(topDeferredDomainsHandler{dashboard}))
	mux.Handle("/api/v0/deliveryStatus", chain.WithEndpoint(deliveryStatusHandler{dashboard}))
	mux.Handle("/api/v0/appVersion", chain.WithError(appVersionHandler{}))
}
