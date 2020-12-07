package v2

import (
	"encoding/gob"
	"errors"
	"fmt"
	"github.com/gorilla/sessions"
	"gitlab.com/lightmeter/controlcenter/auth"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/httputil"
	"log"
	"mime"
	"net/http"
	"os"
	"path"
	"time"
)

type CookieStoreRegistrar struct {
	*auth.Auth
	workspaceDirectory string
}

const SessionDuration = time.Hour * 24 * 7 // 1 week

func (r *CookieStoreRegistrar) CookieStore() sessions.Store {
	sessionsDir := path.Join(r.workspaceDirectory, "http_sessions")
	errorutil.MustSucceed(os.MkdirAll(sessionsDir, os.ModePerm), "Creating http sessions directory")
	store := sessions.NewFilesystemStore(sessionsDir, r.Auth.SessionKeys()...)
	store.Options.HttpOnly = true
	store.Options.MaxAge = int(SessionDuration.Seconds())
	store.Options.SameSite = http.SameSiteStrictMode

	return store
}

func NewAuthenticator(auth *auth.Auth, workspaceDirectory string) *Authenticator {
	return NewAuthenticatorWithOptions(
		&CookieStoreRegistrar{
			Auth:               auth,
			workspaceDirectory: workspaceDirectory,
		},
	)
}

type SessionData struct {
	Email string
	Name  string
}

func (s SessionData) IsAuthenticated() bool {
	return len(s.Email) > 0
}

func init() {
	gob.Register(&SessionData{})
}

type Authenticator struct {
	auth  auth.Registrar
	Store sessions.Store
}

type RegistrarCookieStore interface {
	auth.Registrar
	CookieStore() sessions.Store
}

func NewAuthenticatorWithOptions(
	registrar RegistrarCookieStore,
) *Authenticator {
	return &Authenticator{registrar, registrar.CookieStore()}
}

const SessionName = "controlcenter"

type loginResponse struct {
	Error string
}

func handleForm(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodPost {
		return fmt.Errorf("Error http method mismatch: %v", r.Method)
	}

	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		return fmt.Errorf("Error parse media type: %v", err)
	}

	if mediaType != "application/x-www-form-urlencoded" {
		return fmt.Errorf("Error media type mismatch: %v", err)
	}

	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("Error parse form: %v", err)
	}

	return nil
}

func HandleLogin(auth *Authenticator, w http.ResponseWriter, r *http.Request) {
	if err := handleForm(w, r); err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	email := r.Form.Get("email")
	password := r.Form.Get("password")

	authOk, userData, err := auth.auth.Authenticate(r.Context(), email, password)

	if err != nil {
		log.Println("Error on authentication:", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	if !authOk {
		if err := httputil.WriteJson(w, loginResponse{Error: "Invalid email address or password"}, http.StatusUnauthorized); err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		return
	}

	session, err := auth.Store.New(r, SessionName)
	if err != nil {
		log.Println("Error creating new session:", errorutil.Wrap(err))
	}

	session.Values["auth"] = SessionData{Email: email, Name: userData.Name}

	if err := session.Save(r, w); err != nil {
		log.Println("Error saving session on Login:", errorutil.Wrap(err))
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	if err := httputil.WriteJson(w, loginResponse{Error: ""}, http.StatusOK); err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}
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
	if !hasEmail || len(email) == 0 {
		return "", "", "", errors.New("email is missing")
	}

	password, hasPassword := r.Form["password"]
	if !hasPassword || len(password) == 0 {
		return "", "", "", errors.New("password is missing")
	}

	name, hasName := r.Form["name"]
	if !hasName || len(name) == 0 {
		return "", "", "", errors.New("name is missing")
	}

	return email[0], name[0], password[0], nil
}

func HandleRegistration(auth *Authenticator, w http.ResponseWriter, r *http.Request) {
	if err := handleForm(w, r); err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	email, name, password, err := extractRegistrationFormInfo(r)

	if err != nil {
		log.Println("Registration error:", err)
		w.WriteHeader(http.StatusBadRequest)

		return
	}

	if err := auth.auth.Register(r.Context(), email, name, password); err != nil {
		handleRegistrationFailure(err, w, r)

		return
	}

	session, err := auth.Store.New(r, SessionName)
	if err != nil {
		log.Println("Error creating new session:", errorutil.Wrap(err))
	}

	// Implicitly log in
	session.Values["auth"] = SessionData{Email: email, Name: name}
	if err := session.Save(r, w); err != nil {
		log.Println("Error saving session on Login:", errorutil.Wrap(err))
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	if err := httputil.WriteJson(w, registrationResponse{}, http.StatusOK); err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}
}

func HandleLogout(w http.ResponseWriter, r *http.Request, session *sessions.Session) {
	sessionData, ok := session.Values["auth"].(*SessionData)
	if !(ok && sessionData.IsAuthenticated()) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	log.Println("User", sessionData.Email, "logs out")

	session.Values["auth"] = nil

	if err := session.Save(r, w); err != nil {
		log.Println("Error saving session on Login:", errorutil.Wrap(err))
		w.WriteHeader(http.StatusInternalServerError)

		return
	}
}

// do not redirect to any page
func IsNotLoginOrNotRegistered(auth *Authenticator, w http.ResponseWriter, r *http.Request) {
	hasAnyUser, err := auth.auth.HasAnyUser(r.Context())
	if err != nil {
		log.Println("Error check has any users: ", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	if !hasAnyUser {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	session, err := auth.Store.Get(r, SessionName)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	sessionData, ok := session.Values["auth"].(*SessionData)
	if !(ok && sessionData.IsAuthenticated()) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	w.WriteHeader(http.StatusOK)
}
