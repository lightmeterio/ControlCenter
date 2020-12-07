package httpauth

import (
	"gitlab.com/lightmeter/controlcenter/httpauth/auth"
	"gitlab.com/lightmeter/controlcenter/httpmiddleware"
	"log"
	"net/http"
)

func HttpAuthenticator(mux *http.ServeMux, a *auth.Authenticator) {
	chain := httpmiddleware.WithDefaultStackWithoutAuth()
	mux.Handle("/auth/check", chain.WithEndpoint(httpmiddleware.CustomHTTPHandler(func(w http.ResponseWriter, r *http.Request) error {
		auth.IsNotLoginOrNotRegistered(a, w, r)
		return nil
	})))

	mux.Handle("/login", chain.WithEndpoint(httpmiddleware.CustomHTTPHandler(func(w http.ResponseWriter, r *http.Request) error {
		auth.HandleLogin(a, w, r)
		return nil
	})))

	mux.Handle("/logout", chain.WithEndpoint(httpmiddleware.CustomHTTPHandler(func(w http.ResponseWriter, r *http.Request) error {
		session, err := a.Store.Get(r, auth.SessionName)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			log.Println(err)
			return nil
		}
		auth.HandleLogout(w, r, session)
		return nil
	})))

	mux.Handle("/register", chain.WithEndpoint(httpmiddleware.CustomHTTPHandler(func(w http.ResponseWriter, r *http.Request) error {
		auth.HandleRegistration(a, w, r)
		return nil
	})))
}
