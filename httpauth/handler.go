package httpauth

import (
	"github.com/gorilla/sessions"
	"gitlab.com/lightmeter/controlcenter/auth"
	"gitlab.com/lightmeter/controlcenter/util"
	"net/http"
	"net/url"
	"os"
	"path"
	"time"
)

func changeRequestURL(r *http.Request, path string) *http.Request {
	newReq := &http.Request{}
	*newReq = *r
	newReq.URL = new(url.URL)
	newReq.URL.Path = path
	return newReq
}

type CookieStoreRegistrar struct {
	*auth.Auth
	workspaceDirectory string
}

const SessionDuration = time.Hour * 24 * 7 // 1 week

func (r *CookieStoreRegistrar) CookieStore() sessions.Store {
	sessionsDir := path.Join(r.workspaceDirectory, "http_sessions")
	util.MustSucceed(os.MkdirAll(sessionsDir, os.ModePerm), "Creating http sessions directory")
	store := sessions.NewFilesystemStore(sessionsDir, r.Auth.SessionKeys()...)
	store.Options.HttpOnly = true
	store.Options.MaxAge = int(SessionDuration.Seconds())
	store.Options.SameSite = http.SameSiteStrictMode
	return store
}

func NewAuthenticator(h http.Handler, auth *auth.Auth, workspaceDirectory string, publicPaths []string) *Authenticator {
	return NewAuthenticatorWithOptions(
		AuthHandlers{
			Unauthorized: func(w http.ResponseWriter, r *http.Request) {
			},
			Public: func(w http.ResponseWriter, r *http.Request) {
				h.ServeHTTP(w, r)
			},
			ShowLogin: func(w http.ResponseWriter, r *http.Request) {
				h.ServeHTTP(w, changeRequestURL(r, "/login.i18n.html"))
			},
			Register: func(w http.ResponseWriter, r *http.Request) {
				h.ServeHTTP(w, changeRequestURL(r, "/register.i18n.html"))
			},
			LoginFailure: func(w http.ResponseWriter, r *http.Request) {
			},
			SecretArea: func(session SessionData, w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Cache-Control", "no-store")
				h.ServeHTTP(w, r)
			},
			Logout: func(session SessionData, w http.ResponseWriter, r *http.Request) {
			},
			ServerError: func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(`Some Internal Error Happened :-(`))
			},
		},
		&CookieStoreRegistrar{
			Auth:               auth,
			workspaceDirectory: workspaceDirectory,
		},
		publicPaths,
	)
}

type handler struct {
	*Authenticator
}

func Serve(h http.Handler, auth *auth.Auth, workspaceDirectory string, public []string) *handler {
	return &handler{NewAuthenticator(h, auth, workspaceDirectory, public)}
}
