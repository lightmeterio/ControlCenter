package httpauth

import (
	"gitlab.com/lightmeter/controlcenter/httpauth/auth"
	"gitlab.com/lightmeter/controlcenter/httpmiddleware"
	"net/http"
)

func HttpAuthenticator(mux *http.ServeMux, a *auth.Authenticator) {
	chain := httpmiddleware.WithDefaultStackWithoutAuth()
	mux.Handle("/auth/check", chain.WithEndpoint(httpmiddleware.CustomHTTPHandler(func(w http.ResponseWriter, r *http.Request) error {
		return auth.IsNotLoginOrNotRegistered(a, w, r)
	})))

	mux.Handle("/login", chain.WithEndpoint(httpmiddleware.CustomHTTPHandler(func(w http.ResponseWriter, r *http.Request) error {
		return auth.HandleLogin(a, w, r)
	})))

	mux.Handle("/logout", chain.WithEndpoint(httpmiddleware.CustomHTTPHandler(func(w http.ResponseWriter, r *http.Request) error {
		session, err := a.Store.Get(r, auth.SessionName)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return nil
		}
		return auth.HandleLogout(w, r, session)
	})))

	mux.Handle("/register", chain.WithEndpoint(httpmiddleware.CustomHTTPHandler(func(w http.ResponseWriter, r *http.Request) error {
		return auth.HandleRegistration(a, w, r)
	})))
}
