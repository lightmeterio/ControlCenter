// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package api

import (
	"errors"
	"gitlab.com/lightmeter/controlcenter/detective"
	"gitlab.com/lightmeter/controlcenter/detective/escalator"
	"gitlab.com/lightmeter/controlcenter/httpauth/auth"
	httpauth "gitlab.com/lightmeter/controlcenter/httpauth/auth"
	"gitlab.com/lightmeter/controlcenter/httpmiddleware"
	"gitlab.com/lightmeter/controlcenter/meta"
	"gitlab.com/lightmeter/controlcenter/pkg/httperror"
	detectivesettings "gitlab.com/lightmeter/controlcenter/settings/detective"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/httputil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"net/http"
	"strconv"
	"time"
)

func requireDetectiveAuth(auth *httpauth.Authenticator, settingsReader *meta.Reader) httpmiddleware.Middleware {
	return func(h httpmiddleware.CustomHTTPHandler) httpmiddleware.CustomHTTPHandler {
		return httpmiddleware.CustomHTTPHandler(func(w http.ResponseWriter, r *http.Request) error {
			/* The detective handler can be accessed if authenticated
			 * or if the 'open to end users' setting is enabled.
			 */

			settings := detectivesettings.Settings{}
			err := settingsReader.RetrieveJson(r.Context(), detectivesettings.SettingKey, &settings)
			if err != nil && !errors.Is(err, meta.ErrNoSuchKey) {
				return httperror.NewHTTPStatusCodeError(http.StatusInternalServerError, errorutil.Wrap(err))
			}

			if settings.EndUsersEnabled {
				if err := h.ServeHTTP(w, r); err != nil {
					return httperror.NewHTTPStatusCodeError(http.StatusInternalServerError, errorutil.Wrap(err))
				}

				return nil
			}

			// If setting is disabled, user must be authenticated
			sessionData, err := httpauth.GetSessionData(auth, r)
			if err != nil {
				return httperror.NewHTTPStatusCodeError(http.StatusUnauthorized, errorutil.Wrap(err))
			}

			if !sessionData.IsAuthenticated() {
				return httperror.NewHTTPStatusCodeError(http.StatusUnauthorized, httpauth.ErrUnauthenticated)
			}

			if err := h.ServeHTTP(w, r); err != nil {
				return httperror.NewHTTPStatusCodeError(http.StatusInternalServerError, errorutil.Wrap(err))
			}

			return nil
		})
	}
}

type detectiveHandler struct {
	//nolint:structcheck
	detective detective.Detective
}

type checkMessageDeliveryHandler detectiveHandler
type Interval string

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
	if r.ParseForm() != nil {
		return httperror.NewHTTPStatusCodeError(http.StatusUnprocessableEntity, errors.New("Wrong Input"))
	}

	interval, err := timeutil.ParseTimeInterval(r.Form.Get("from"), r.Form.Get("to"), time.UTC)
	if err != nil {
		return httperror.NewHTTPStatusCodeError(http.StatusUnprocessableEntity, err)
	}

	page, err := strconv.Atoi(r.Form.Get("page"))
	if err != nil {
		return httperror.NewHTTPStatusCodeError(http.StatusUnprocessableEntity, err)
	}

	messages, err := h.detective.CheckMessageDelivery(r.Context(), r.Form.Get("mail_from"), r.Form.Get("mail_to"), interval, page)

	if err != nil {
		return httperror.NewHTTPStatusCodeError(http.StatusUnprocessableEntity, err)
	}

	return httputil.WriteJson(w, messages, http.StatusOK)
}

func HttpDetective(auth *auth.Authenticator, mux *http.ServeMux, timezone *time.Location, detective detective.Detective, settingsReader *meta.Reader) {
	mux.Handle("/api/v0/checkMessageDeliveryStatus", httpmiddleware.New(
		httpmiddleware.RequestWithTimeout(httpmiddleware.DefaultTimeout), requireDetectiveAuth(auth, settingsReader),
	).WithEndpoint(checkMessageDeliveryHandler{detective}))
}

type detectiveEscalatorHandler struct {
	requester escalator.Requester
	detective detective.Detective
}

// @Summary Escalate Message
// @Param mail_from      query string true "Sender email address"
// @Param mail_to        query string true "Recipient email address"
// @Param timestamp_from query string true "Initial timestamp in the format 1999-12-23 12:00:00"
// @Param timestamp_to   query string true "Final timestamp in the format 1999-12-23 14:00:00"
// @Router /api/v0/escalateMessage [post]
func (h detectiveEscalatorHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return httperror.NewHTTPStatusCodeError(http.StatusBadRequest, err)
	}

	interval, err := timeutil.ParseTimeInterval(r.Form.Get("from"), r.Form.Get("to"), time.UTC)
	if err != nil {
		return httperror.NewHTTPStatusCodeError(http.StatusBadRequest, err)
	}

	if err := escalator.TryToEscalateRequest(r.Context(), h.detective, h.requester, r.Form.Get("mail_from"), r.Form.Get("mail_to"), interval); err != nil {
		return httperror.NewHTTPStatusCodeError(http.StatusInternalServerError, err)
	}

	return httputil.WriteJson(w, "ok", http.StatusOK)
}
