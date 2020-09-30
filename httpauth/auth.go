package httpauth

import (
	"encoding/gob"
	"errors"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/httputil"
	"log"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/sessions"
	"gitlab.com/lightmeter/controlcenter/auth"
)

type SessionData struct {
	Email string
	Name  string
}

func (s SessionData) isAuthenticated() bool {
	return len(s.Email) > 0
}

func init() {
	gob.Register(&SessionData{})
}

type AuthHandlers struct {
	Unauthorized func(http.ResponseWriter, *http.Request)
	Public       func(http.ResponseWriter, *http.Request)
	ShowLogin    func(http.ResponseWriter, *http.Request)
	Register     func(http.ResponseWriter, *http.Request)
	LoginFailure func(http.ResponseWriter, *http.Request)
	SecretArea   func(SessionData, http.ResponseWriter, *http.Request)
	Logout       func(SessionData, http.ResponseWriter, *http.Request)
	ServerError  func(http.ResponseWriter, *http.Request)
}

type Authenticator struct {
	handlers AuthHandlers
	auth     auth.Registrar
	store    sessions.Store
	public   []string
}

type RegistrarCookieStore interface {
	auth.Registrar
	CookieStore() sessions.Store
}

func NewAuthenticatorWithOptions(
	handlers AuthHandlers,
	registrar RegistrarCookieStore,
	publicPaths []string,
) *Authenticator {
	return &Authenticator{handlers, registrar, registrar.CookieStore(), publicPaths}
}

func acceptOnlyGetOrPost(auth *Authenticator, w http.ResponseWriter, r *http.Request, getCb, postCb func()) {
	if r.Method == "GET" {
		getCb()
		return
	}

	if r.Method != "POST" {
		log.Println("Invalid HTTP method")
		w.WriteHeader(http.StatusInternalServerError)
		auth.handlers.ServerError(w, r)

		return
	}

	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))

	if err != nil {
		log.Println("Error parsing form mime type:", err)
		w.WriteHeader(http.StatusInternalServerError)
		auth.handlers.ServerError(w, r)

		return
	}

	if mediaType != "application/x-www-form-urlencoded" {
		log.Println("Invalid mime type")
		w.WriteHeader(http.StatusInternalServerError)
		auth.handlers.ServerError(w, r)

		return
	}

	if err := r.ParseForm(); err != nil {
		log.Println("Failed parsing form")
		w.WriteHeader(http.StatusInternalServerError)
		auth.handlers.ServerError(w, r)

		return
	}

	postCb()
}

const sessionName = "lightmeter-controlcenter-session"

func handleFailureOnObtainingSession(auth *Authenticator, w http.ResponseWriter, r *http.Request) {
	// replace session cookie by a empty one, which means to delete it
	cookie := &http.Cookie{
		Name:     sessionName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
	}

	redirectUrl, err := defaultUnauthorisedRedirectUrl(auth, w, r)

	if err != nil {
		log.Println("Error getting session:", err)
		w.WriteHeader(http.StatusInternalServerError)
		auth.handlers.ServerError(w, r)

		return
	}

	http.SetCookie(w, cookie)

	http.Redirect(w, r, redirectUrl, http.StatusFound)
}

func withSession(auth *Authenticator, w http.ResponseWriter, r *http.Request, cb func(*sessions.Session)) {
	session, err := auth.store.Get(r, sessionName)

	if err != nil {
		log.Println("Error getting session. Force session expiration:", err)
		handleFailureOnObtainingSession(auth, w, r)

		return
	}

	cb(session)
}

type loginResponse struct {
	Error string
}

func handleLoginSubmission(auth *Authenticator, w http.ResponseWriter, r *http.Request, session *sessions.Session) {
	email := r.Form.Get("email")
	password := r.Form.Get("password")

	authOk, userData, err := auth.auth.Authenticate(email, password)

	if err != nil {
		log.Println("Error on authentication:", err)
		w.WriteHeader(http.StatusInternalServerError)
		auth.handlers.ServerError(w, r)

		return
	}

	if !authOk {
		if err := httputil.WriteJson(w, loginResponse{Error: "Invalid email address or password"}, http.StatusUnauthorized); err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		auth.handlers.LoginFailure(w, r)

		return
	}

	session.Values["auth"] = SessionData{Email: email, Name: userData.Name}

	if err := saveSession(auth, w, r, session); err != nil {
		log.Println("Error saving session on Login:", err)
		return
	}

	if err := httputil.WriteJson(w, loginResponse{Error: ""}, http.StatusOK); err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}
}

func handleFormSubmissionOnSession(
	auth *Authenticator,
	w http.ResponseWriter,
	r *http.Request,
	getMethodHandler func(w http.ResponseWriter, r *http.Request),
	postMethodHandler func(auth *Authenticator, w http.ResponseWriter, r *http.Request, session *sessions.Session),
) {
	withSession(auth, w, r, func(session *sessions.Session) {
		if sessionData, ok := session.Values["auth"].(*SessionData); ok && sessionData.isAuthenticated() {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}

		acceptOnlyGetOrPost(auth, w, r, func() {
			getMethodHandler(w, r)
		}, func() {
			postMethodHandler(auth, w, r, session)
		})
	})
}

func handleLogin(auth *Authenticator, w http.ResponseWriter, r *http.Request) {
	handleFormSubmissionOnSession(auth, w, r, auth.handlers.ShowLogin, handleLoginSubmission)
}

func saveSession(auth *Authenticator, w http.ResponseWriter, r *http.Request, session *sessions.Session) error {
	if err := session.Save(r, w); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		auth.handlers.ServerError(w, r)

		return errorutil.Wrap(err)
	}

	return nil
}

type registrationResponse struct {
	Error    string
	Detailed interface{}
}

func handleRegistrationFailure(err error, w http.ResponseWriter, r *http.Request) {
	response := registrationResponse{
		Error: errorutil.TryToUnwrap(err).Error(),

		Detailed: func() interface{} {
			if e, ok := errorutil.ErrorAs(err, &auth.PasswordValidationError{}); ok {
				d, _ := e.(*auth.PasswordValidationError)
				return &d.Result
			}

			return nil
		}(),
	}

	if err := httputil.WriteJson(w, response, http.StatusUnauthorized); err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}
}

func extractRegistrationFormInfo(r *http.Request) (string, string, string, error) {
	email, hasEmail := r.Form["email"]
	password, hasPassword := r.Form["password"]
	name := r.Form.Get("name")

	if !(hasEmail && len(email) > 0 && hasPassword && len(password) > 0) {
		return "", "", "", errors.New("Missing form information")
	}

	return email[0], name, password[0], nil
}

func handleRegistrationSubmission(auth *Authenticator, w http.ResponseWriter, r *http.Request, session *sessions.Session) {
	email, name, password, err := extractRegistrationFormInfo(r)

	if err != nil {
		log.Println("Registration error:", err)
		w.WriteHeader(http.StatusInternalServerError)
		auth.handlers.ServerError(w, r)

		return
	}

	if err := auth.auth.Register(email, name, password); err != nil {
		handleRegistrationFailure(err, w, r)
		return
	}

	// Implicitly log in
	session.Values["auth"] = SessionData{Email: email, Name: name}

	if err := saveSession(auth, w, r, session); err != nil {
		log.Println("Error saving session on Registration:", err)
		return
	}

	if err := httputil.WriteJson(w, registrationResponse{}, http.StatusOK); err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}
}

func handleRegistration(auth *Authenticator, w http.ResponseWriter, r *http.Request) {
	handleFormSubmissionOnSession(auth, w, r, auth.handlers.Register, handleRegistrationSubmission)
}

func defaultUnauthorisedRedirectUrl(auth *Authenticator, w http.ResponseWriter, r *http.Request) (string, error) {
	ok, err := auth.auth.HasAnyUser()

	if err != nil {
		return "", errorutil.Wrap(err)
	}

	if ok {
		return "/login", nil
	}

	return "/register", nil
}

func handleUnauthorized(auth *Authenticator, w http.ResponseWriter, r *http.Request) {
	auth.handlers.Unauthorized(w, r)

	redirectUrl, err := defaultUnauthorisedRedirectUrl(auth, w, r)

	if err != nil {
		log.Println("Error Checking whether any user is already registred:", err)
		w.WriteHeader(http.StatusInternalServerError)
		auth.handlers.ServerError(w, r)

		return
	}

	http.Redirect(w, r, redirectUrl, http.StatusFound)
}

func handleSecretArea(session SessionData, auth *Authenticator, w http.ResponseWriter, r *http.Request) {
	auth.handlers.SecretArea(session, w, r)
}

func handleLogout(auth *Authenticator, w http.ResponseWriter, r *http.Request) {
	withSession(auth, w, r, func(session *sessions.Session) {
		sessionData, ok := session.Values["auth"].(*SessionData)

		if !(ok && sessionData.isAuthenticated()) {
			handleUnauthorized(auth, w, r)
			return
		}

		log.Println("User", sessionData.Email, "logs out")

		session.Values["auth"] = nil

		auth.handlers.Logout(*sessionData, w, r)

		if err := saveSession(auth, w, r, session); err != nil {
			log.Println("Error saving session on Logout:", err)
			return
		}

		http.Redirect(w, r, "/login", http.StatusFound)
	})
}

func withBasicHTTPAuth(auth *Authenticator, user, password string, w http.ResponseWriter, r *http.Request) {
	ok, userData, err := auth.auth.Authenticate(user, password)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	sessionData := SessionData{Email: userData.Email, Name: userData.Name}

	auth.handlers.SecretArea(sessionData, w, r)
}

// NOTE: I am quite sure this algorithm for checking url prefixes has
// already implementing somewhere in some routing libraries
// but that's okay for now, and I don't believe it'll be bad performance-wise
// as the list of public paths is usually small and no dynamic memory allocation
// is being performed.
// It behaves differntly from the standard http routing in the way that wont't
// match the longest route, which can make our life a bit harder in the future
func pathIsPublic(auth *Authenticator, url *url.URL) bool {
	for _, p := range auth.public {
		if !strings.HasPrefix(url.Path, p) {
			continue
		}

		if len(url.Path) == len(p) {
			return true
		}

		// here we know that len(url.Path) > len(p)
		if url.Path[len(p)] == '/' {
			return true
		}
	}

	return false
}

func handlePublic(auth *Authenticator, w http.ResponseWriter, r *http.Request) {
	auth.handlers.Public(w, r)
}

func (auth *Authenticator) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if pathIsPublic(auth, r.URL) {
		handlePublic(auth, w, r)
		return
	}

	if r.URL.Path == "/login" {
		handleLogin(auth, w, r)
		return
	}

	if r.URL.Path == "/register" {
		handleRegistration(auth, w, r)
		return
	}

	if r.URL.Path == "/logout" {
		handleLogout(auth, w, r)
		return
	}

	if user, password, ok := r.BasicAuth(); ok {
		withBasicHTTPAuth(auth, user, password, w, r)
		return
	}

	withSession(auth, w, r, func(session *sessions.Session) {
		sessionData, ok := session.Values["auth"].(*SessionData)

		if !(ok && sessionData.isAuthenticated()) {
			handleUnauthorized(auth, w, r)
			return
		}

		handleSecretArea(*sessionData, auth, w, r)
	})
}
