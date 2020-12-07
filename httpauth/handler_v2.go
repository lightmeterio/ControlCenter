package httpauth

import (
	v2 "gitlab.com/lightmeter/controlcenter/httpauth/v2"
	"gitlab.com/lightmeter/controlcenter/httpmiddleware"
	"log"
	"net/http"
)

func HttpAuthenticator(mux *http.ServeMux, auth *v2.Authenticator) {
	chain := httpmiddleware.WithDefaultStack()
	mux.Handle("/auth/check", chain.WithEndpoint(httpmiddleware.CustomHTTPHandler(func(w http.ResponseWriter, r *http.Request) error {
		v2.IsNotLoginOrNotRegistered(auth, w, r)
		return nil
	})))

	mux.Handle("/login", chain.WithEndpoint(httpmiddleware.CustomHTTPHandler(func(w http.ResponseWriter, r *http.Request) error {
		v2.HandleLogin(auth, w, r)
		return nil
	})))

	mux.Handle("/logout", chain.WithEndpoint(httpmiddleware.CustomHTTPHandler(func(w http.ResponseWriter, r *http.Request) error {
		session, err := auth.Store.Get(r, v2.SessionName)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			log.Println(err)
			return nil
		}
		v2.HandleLogout(w, r, session)
		return nil
	})))

	mux.Handle("/register", chain.WithEndpoint(httpmiddleware.CustomHTTPHandler(func(w http.ResponseWriter, r *http.Request) error {
		v2.HandleRegistration(auth, w, r)
		return nil
	})))
}
