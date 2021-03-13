// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package api

import (
	httpauth "gitlab.com/lightmeter/controlcenter/httpauth/auth"
	"gitlab.com/lightmeter/controlcenter/httpmiddleware"
	"gitlab.com/lightmeter/controlcenter/insights/core"
	"gitlab.com/lightmeter/controlcenter/pkg/httperror"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/httputil"
	"net/http"
)

// allows something to be seen only if before the user registration ends or the user is authenticated
// TODO: implement unit tests for this function
// TODO: maybe move it to a more suitable place, as it could be useful to more use cases
func requireAuthenticationOnlyAfterSystemHasAnyUser(auth *httpauth.Authenticator) httpmiddleware.Middleware {
	return func(h httpmiddleware.CustomHTTPHandler) httpmiddleware.CustomHTTPHandler {
		return httpmiddleware.CustomHTTPHandler(func(w http.ResponseWriter, r *http.Request) error {
			// the progress endpoint can only be accessed in case the user is authenticated
			// or the system registration has not finished yet (aka. no users are registred)
			hasAnyUser, err := auth.Registrar.HasAnyUser(r.Context())
			if err != nil {
				return httperror.NewHTTPStatusCodeError(http.StatusInternalServerError, errorutil.Wrap(err))
			}

			if !hasAnyUser {
				// still pre-registration. Go ahead with the request
				if err := h.ServeHTTP(w, r); err != nil {
					return httperror.NewHTTPStatusCodeError(http.StatusInternalServerError, errorutil.Wrap(err))
				}

				return nil
			}

			// Here the user must be authenticated
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

func HttpInsightsProgress(auth *httpauth.Authenticator, mux *http.ServeMux, p core.ProgressFetcher) {
	mux.Handle("/api/v0/importProgress", httpmiddleware.New(
		httpmiddleware.RequestWithTimeout(httpmiddleware.DefaultTimeout), requireAuthenticationOnlyAfterSystemHasAnyUser(auth),
	).WithEndpoint(importProgressHandler{f: p}))
}

type importProgressHandler struct {
	f core.ProgressFetcher
}

// @Summary Fetch Insights
// @Produce json
// @Success 200 {object} core.Progress
// @Failure 422 {string} string "desc"
// @Router /api/v0/importProgress [get]
func (h importProgressHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) error {
	p, err := h.f.Progress(r.Context())
	if err != nil {
		return errorutil.Wrap(err, "Error obtaining import progress")
	}

	if err := httputil.WriteJson(w, p, http.StatusOK); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}
