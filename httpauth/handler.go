// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package httpauth

import (
	"gitlab.com/lightmeter/controlcenter/httpauth/auth"
	"gitlab.com/lightmeter/controlcenter/httpmiddleware"
	"net/http"
)

func HttpAuthenticator(mux *http.ServeMux, a *auth.Authenticator) {
	unauthenticated := httpmiddleware.WithDefaultStackWithoutAuth()

	mux.Handle("/auth/check", unauthenticated.WithEndpoint(httpmiddleware.CustomHTTPHandler(func(w http.ResponseWriter, r *http.Request) error {
		return auth.IsNotLoginOrNotRegistered(a, w, r)
	})))

	mux.Handle("/login", unauthenticated.WithEndpoint(httpmiddleware.CustomHTTPHandler(func(w http.ResponseWriter, r *http.Request) error {
		return auth.HandleLogin(a, w, r)
	})))

	mux.Handle("/api/v0/userInfo", unauthenticated.WithEndpoint(httpmiddleware.CustomHTTPHandler(func(w http.ResponseWriter, r *http.Request) error {
		return auth.HandleGetUserData(a, w, r)
	})))

	mux.Handle("/logout", unauthenticated.WithEndpoint(httpmiddleware.CustomHTTPHandler(func(w http.ResponseWriter, r *http.Request) error {
		session, err := a.Store.Get(r, auth.SessionName)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return nil
		}

		return auth.HandleLogout(w, r, session)
	})))

	mux.Handle("/register", unauthenticated.WithEndpoint(httpmiddleware.CustomHTTPHandler(func(w http.ResponseWriter, r *http.Request) error {
		return auth.HandleRegistration(a, w, r)
	})))
}
