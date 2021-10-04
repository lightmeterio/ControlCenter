// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package api

import (
	"gitlab.com/lightmeter/controlcenter/connectionstats"
	"gitlab.com/lightmeter/controlcenter/httpauth/auth"
	"gitlab.com/lightmeter/controlcenter/httpmiddleware"
	"gitlab.com/lightmeter/controlcenter/pkg/httperror"
	"gitlab.com/lightmeter/controlcenter/util/httputil"
	"net/http"
	"time"
)

type smtpAuthAttemptsHandler struct {
	accessor *connectionstats.Accessor
}

// @Summary Information Fetch SMTP authentication attempts
// @Param from query string true "Initial date in the format 1999-12-23"
// @Param to   query string true "Final date in the format 1999-12-23"
// @Produce json
// @Success 200 {object} connectionstats.Stats "desc"
// @Failure 422 {string} string "desc"
// @Router /api/v0/fetchSmtpAuthAttempts [get]
func (handler smtpAuthAttemptsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) error {
	interval := httpmiddleware.GetIntervalFromContext(r)

	result, err := handler.accessor.FetchAuthAttempts(r.Context(), interval)
	if err != nil {
		return httperror.NewHTTPStatusCodeError(http.StatusInternalServerError, err)
	}

	return httputil.WriteJson(w, result, http.StatusOK)
}

func HttpConnectionsDashboard(auth *auth.Authenticator, mux *http.ServeMux, timezone *time.Location, accessor *connectionstats.Accessor) {
	authenticated := httpmiddleware.WithDefaultStack(auth, httpmiddleware.RequestWithInterval(timezone))
	mux.Handle("/api/v0/fetchSmtpAuthAttempts", authenticated.WithEndpoint(smtpAuthAttemptsHandler{accessor: accessor}))
}
