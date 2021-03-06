// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package httpauth

import (
	"gitlab.com/lightmeter/controlcenter/httpauth/auth"
	"gitlab.com/lightmeter/controlcenter/httpmiddleware"
	"gitlab.com/lightmeter/controlcenter/metadata"
	"net/http"
	"time"
)

func HttpAuthenticator(mux *http.ServeMux, a *auth.Authenticator, settingsReader metadata.Reader, isBehindReverseProxy bool) {
	/* Endpoint /auth/check is constantly called in the UI:
	 * - every 5sec once the user is logged (js interval)
	 * - once on every page change (user action)
	 *   (when the page changes, the js interval is reset)
	 * So the limit in one minute can't be less than
	 *   60/5 = 12 (js interval)
	 *   or
	 *   1 user call per second = 60 per minute (clicking frenzy by the user = js interval never triggers)
	 * => rate-limit to 60 calls per minute
	 */
	unauthenticatedAndRateLimitedForFrequentCalls := httpmiddleware.WithDefaultStackWithoutAuth(
		httpmiddleware.RequestWithRateLimit(1*time.Minute, 60, isBehindReverseProxy, httpmiddleware.BlockQuery),
	)

	mux.Handle("/auth/check", unauthenticatedAndRateLimitedForFrequentCalls.
		WithEndpoint(httpmiddleware.CustomHTTPHandler(func(w http.ResponseWriter, r *http.Request) error {
			return auth.IsNotLoginOrNotRegistered(a, w, r)
		})))

	unauthenticatedAndRateLimited := httpmiddleware.WithDefaultStackWithoutAuth(
		httpmiddleware.RequestWithRateLimit(5*time.Minute, 20, isBehindReverseProxy, httpmiddleware.BlockQuery),
	)

	mux.Handle("/auth/detective", unauthenticatedAndRateLimited.
		WithEndpoint(httpmiddleware.CustomHTTPHandler(func(w http.ResponseWriter, r *http.Request) error {
			return auth.IsNotLoginAndNotEndUsersEnabled(a, w, r, settingsReader)
		})))

	mux.Handle("/login", unauthenticatedAndRateLimited.WithEndpoint(
		httpmiddleware.CustomHTTPHandler(
			func(w http.ResponseWriter, r *http.Request) error {
				return auth.HandleLogin(a, w, r)
			})))

	// NOTE: This endpoint is actually authenticated, see auth.HandleGetUserSystemData
	// NOTE: use a regular 'unauthenticated' endpoint, with no rate-limiting, since /userInfo is actually authenticated
	// TODO: such behaviour should be explicit in the implementation, requiring an authenticated
	// middleware
	mux.Handle("/api/v0/userInfo", httpmiddleware.WithDefaultStackWithoutAuth().
		WithEndpoint(httpmiddleware.CustomHTTPHandler(func(w http.ResponseWriter, r *http.Request) error {
			return auth.HandleGetUserSystemData(a, settingsReader, w, r)
		})))

	mux.Handle("/logout", unauthenticatedAndRateLimited.
		WithEndpoint(httpmiddleware.CustomHTTPHandler(func(w http.ResponseWriter, r *http.Request) error {
			session, err := a.Store.Get(r, auth.SessionName)
			if err != nil {
				w.WriteHeader(http.StatusUnauthorized)
				//nolint:nilerr
				return nil
			}

			return auth.HandleLogout(w, r, session)
		})))

	mux.Handle("/register", unauthenticatedAndRateLimited.
		WithEndpoint(httpmiddleware.CustomHTTPHandler(func(w http.ResponseWriter, r *http.Request) error {
			return auth.HandleRegistration(a, w, r)
		})))
}
