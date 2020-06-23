package httpauth

import (
	"github.com/gorilla/sessions"
	"gitlab.com/lightmeter/controlcenter/auth"
	"gitlab.com/lightmeter/controlcenter/util"
	"net/http"
	"net/url"
	"os"
	"path"
)

func changeRequestURL(r *http.Request, path string) *http.Request {
	newReq := &http.Request{}
	newReq.URL = new(url.URL)
	newReq.URL.Path = path
	return newReq
}

type CookieStoreRegistrar struct {
	*auth.Auth
	workspaceDirectory string
}

func (r *CookieStoreRegistrar) CookieStore() sessions.Store {
	sessionsDir := path.Join(r.workspaceDirectory, "http_sessions")
	util.MustSucceed(os.MkdirAll(sessionsDir, os.ModePerm), "Creating http sessions directory")
	store := sessions.NewFilesystemStore(sessionsDir, r.Auth.SessionKeys()...)
	store.Options.HttpOnly = true
	store.Options.MaxAge = 1 * 60 * 60 // cookies last one hour
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
				h.ServeHTTP(w, changeRequestURL(r, "/login.html"))
			},
			Register: func(w http.ResponseWriter, r *http.Request) {
				h.ServeHTTP(w, changeRequestURL(r, "/register.html"))
			},
			LoginFailure: func(w http.ResponseWriter, r *http.Request) {
			},
			SecretArea: func(session SessionData, w http.ResponseWriter, r *http.Request) {
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
	auth *Authenticator
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.auth.ServeHTTP(w, r)
}

func Serve(h http.Handler, auth *auth.Auth, workspaceDirectory string, public []string) *handler {
	return &handler{auth: NewAuthenticator(h, auth, workspaceDirectory, public)}
}
