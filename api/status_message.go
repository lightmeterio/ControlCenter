// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package api

import (
	"gitlab.com/lightmeter/controlcenter/httpauth/auth"
	"gitlab.com/lightmeter/controlcenter/httpmiddleware"
	"gitlab.com/lightmeter/controlcenter/intel/collector"
	"gitlab.com/lightmeter/controlcenter/intel/receptor"
	"gitlab.com/lightmeter/controlcenter/pkg/httperror"
	"gitlab.com/lightmeter/controlcenter/util/httputil"
	"net/http"
)

type statusMessageHandler struct {
	accessor *collector.Accessor
}

type MessageNotification struct {
	ID           string                        `json:"id"`
	Notification *receptor.MessageNotification `json:"notification,omitempty"`
}

// @Summary
// @Success 200 {object} MessageNotification "desc"
// @Failure 422 {string} string "desc"
// @Failure 500 {string} string "desc"
// @Router /api/v0/intelstatus [post]
func (h statusMessageHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return httperror.NewHTTPStatusCodeError(http.StatusBadRequest, err)
	}

	statusMessage, err := h.accessor.GetStatusMessage(r.Context())
	if err != nil {
		return httperror.NewHTTPStatusCodeError(http.StatusInternalServerError, err)
	}

	if statusMessage == nil {
		return httputil.WriteJson(w, statusMessage, http.StatusOK)
	}

	return httputil.WriteJson(w, MessageNotification{ID: statusMessage.ID, Notification: statusMessage.MessageNotification}, http.StatusOK)
}

func HttpStatusMessage(auth *auth.Authenticator, mux *http.ServeMux, accessor *collector.Accessor) {
	mux.Handle("/api/v0/intelstatus",
		httpmiddleware.WithDefaultStack(auth, httpmiddleware.RequestWithTimeout(httpmiddleware.DefaultTimeout)).
			WithEndpoint(statusMessageHandler{accessor: accessor}))
}
