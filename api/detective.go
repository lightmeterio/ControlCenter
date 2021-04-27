// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package api

import (
	"gitlab.com/lightmeter/controlcenter/detective"
	"gitlab.com/lightmeter/controlcenter/httpauth/auth"
	"gitlab.com/lightmeter/controlcenter/httpmiddleware"
	"gitlab.com/lightmeter/controlcenter/pkg/httperror"
	"gitlab.com/lightmeter/controlcenter/util/httputil"
	"net/http"
	"time"
)

type detectiveHandler struct {
	//nolint:structcheck
	detective detective.Detective
}

type checkMessageDeliveryHandler detectiveHandler

// @Summary Check message delivery
// @Param mail_from      query string true "Sender email address"
// @Param mail_to        query string true "Recipient email address"
// @Param timestamp_from query string true "Initial timestamp in the format 1999-12-23 12:00:00"
// @Param timestamp_to   query string true "Final timestamp in the format 1999-12-23 14:00:00"
// @Produce json
// @Success 200 {object} []detective.MessageDelivery "desc"
// @Failure 422 {string} string "desc"
// @Router /api/v0/checkMessageDelivery [get]
func (h checkMessageDeliveryHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) error {
	interval := httpmiddleware.GetIntervalFromContext(r)

	messages, err := h.detective.CheckMessageDelivery(r.Context(), r.Form.Get("mail_from"), r.Form.Get("mail_to"), interval)

	if err != nil {
		return httperror.NewHTTPStatusCodeError(http.StatusUnprocessableEntity, err)
	}

	return httputil.WriteJson(w, messages, http.StatusOK)
}

func HttpDetective(auth *auth.Authenticator, mux *http.ServeMux, timezone *time.Location, detective detective.Detective) {
	// TODO: unauthenticated if allowed in LMCC settings
	// 	unauthenticated := httpmiddleware.WithDefaultStackWithoutAuth()
	// 	mux.Handle("/api/v0/checkMessageDeliveryStatus", unauthenticated.WithEndpoint(appVersionHandler{}))
	authenticated := httpmiddleware.WithDefaultStack(auth, httpmiddleware.RequestWithInterval(timezone))
	mux.Handle("/api/v0/checkMessageDeliveryStatus", authenticated.WithEndpoint(checkMessageDeliveryHandler{detective}))
}
