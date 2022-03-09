// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package api

import (
	"context"
	"gitlab.com/lightmeter/controlcenter/dashboard"
	"gitlab.com/lightmeter/controlcenter/httpauth/auth"
	"gitlab.com/lightmeter/controlcenter/httpmiddleware"
	"gitlab.com/lightmeter/controlcenter/pkg/httperror"
	parser "gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser"
	"gitlab.com/lightmeter/controlcenter/util/httputil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"gitlab.com/lightmeter/controlcenter/version"
	"net/http"
	"strconv"
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
		return httperror.NewHTTPStatusCodeError(http.StatusInternalServerError, err)
	}

	deferred, err := h.dashboard.CountByStatus(r.Context(), parser.DeferredStatus, interval)
	if err != nil {
		return httperror.NewHTTPStatusCodeError(http.StatusInternalServerError, err)
	}

	bounced, err := h.dashboard.CountByStatus(r.Context(), parser.BouncedStatus, interval)
	if err != nil {
		return httperror.NewHTTPStatusCodeError(http.StatusInternalServerError, err)
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
	f func(context.Context, timeutil.TimeInterval) (dashboard.Pairs, error),
	interval timeutil.TimeInterval) error {
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

type trafficBySenderOverTimeHandler struct {
	f func(context.Context, timeutil.TimeInterval, int) (dashboard.MailTrafficPerSenderOverTimeResult, error)
}

// @Summary Messages sent by mailbox over time
// @Param from query string true "Initial date in the format 1999-12-23"
// @Param to   query string true "Final date in the format 1999-12-23"
// @Param granularity query integer 12 "Time granularity in hours"
// @Produce json
// @Success 200 {object} dashboard.MailTrafficPerSenderOverTimeResult
// @Failure 422 {string} string "desc"
// @Router /api/v0/sentMailsByMailbox  [get]

// @Summary Messages bounced by mailbox over time
// @Param from query string true "Initial date in the format 1999-12-23"
// @Param to   query string true "Final date in the format 1999-12-23"
// @Param granularity query integer 12 "Time granularity in hours"
// @Produce json
// @Success 200 {object} dashboard.MailTrafficPerSenderOverTimeResult
// @Failure 422 {string} string "desc"
// @Router /api/v0/bouncedMailsByMailbox  [get]

// @Summary Messages bounced by mailbox over time
// @Param from query string true "Initial date in the format 1999-12-23"
// @Param to   query string true "Final date in the format 1999-12-23"
// @Param granularity query integer 12 "Time granularity in hours"
// @Produce json
// @Success 200 {object} dashboard.MailTrafficPerSenderOverTimeResult
// @Failure 422 {string} string "desc"
// @Router /api/v0/deferredMailsByMailbox  [get]

func (h trafficBySenderOverTimeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) error {
	interval := httpmiddleware.GetIntervalFromContext(r)

	granularity, err := strconv.Atoi(r.Form.Get("granularity"))
	if err != nil {
		return httperror.NewHTTPStatusCodeError(http.StatusUnprocessableEntity, err)
	}

	result, err := h.f(r.Context(), interval, granularity)
	if err != nil {
		return err
	}

	return httputil.WriteJson(w, result, http.StatusOK)
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

func HttpDashboard(auth *auth.Authenticator, mux *http.ServeMux, timezone *time.Location, d dashboard.Dashboard) {
	authenticated := httpmiddleware.WithDefaultStack(auth, httpmiddleware.RequestWithInterval(timezone))
	unauthenticated := httpmiddleware.WithDefaultStackWithoutAuth()

	trafficBySender := map[string]func(context.Context, timeutil.TimeInterval, int) (dashboard.MailTrafficPerSenderOverTimeResult, error){
		"/api/v0/sentMailsByMailbox":     d.SentMailsByMailbox,
		"/api/v0/bouncedMailsByMailbox":  d.BouncedMailsByMailbox,
		"/api/v0/deferredMailsByMailbox": d.DeferredMailsByMailbox,
	}

	for k, v := range trafficBySender {
		mux.Handle(k, authenticated.WithEndpoint(trafficBySenderOverTimeHandler{v}))
	}

	mux.Handle("/api/v0/countByStatus", authenticated.WithEndpoint(countByStatusHandler{d}))
	mux.Handle("/api/v0/topBusiestDomains", authenticated.WithEndpoint(topBusiestDomainsHandler{d}))
	mux.Handle("/api/v0/topBouncedDomains", authenticated.WithEndpoint(topBouncedDomainsHandler{d}))
	mux.Handle("/api/v0/topDeferredDomains", authenticated.WithEndpoint(topDeferredDomainsHandler{d}))
	mux.Handle("/api/v0/deliveryStatus", authenticated.WithEndpoint(deliveryStatusHandler{d}))
	mux.Handle("/api/v0/appVersion", unauthenticated.WithEndpoint(appVersionHandler{}))
}
