// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package api

import (
	"gitlab.com/lightmeter/controlcenter/httpauth/auth"
	"gitlab.com/lightmeter/controlcenter/httpmiddleware"
	"gitlab.com/lightmeter/controlcenter/intel/collector"
	"gitlab.com/lightmeter/controlcenter/pkg/httperror"
	"gitlab.com/lightmeter/controlcenter/util/httputil"
	"net/http"
	"time"
)

type reportsHandler struct {
	intelAccessor *collector.Accessor
}

// @Summary Get latest reports
// @Produce json
// @Success 200 {object} string "desc"
// @Failure 422 {string} string "desc"
// @Router /api/v0/reports [get]
func (h reportsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) error {
	reports, err := h.intelAccessor.GetDispatchedReports(r.Context())

	if err != nil {
		return httperror.NewHTTPStatusCodeError(http.StatusUnprocessableEntity, err)
	}

	return httputil.WriteJson(w, reports, http.StatusOK)
}

func HttpReports(auth *auth.Authenticator, mux *http.ServeMux, timezone *time.Location, intelAccessor *collector.Accessor) {
	authenticated := httpmiddleware.WithDefaultStack(auth, httpmiddleware.RequestWithTimeout(httpmiddleware.DefaultTimeout))

	mux.Handle("/api/v0/reports", authenticated.WithEndpoint(reportsHandler{intelAccessor}))
}
