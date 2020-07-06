package httpsettings

import (
	"errors"
	"gitlab.com/lightmeter/controlcenter/settings"
	"log"
	"mime"
	"net/http"
)

type InitialSetupHandler struct {
	s settings.SystemSetup
}

func NewInitialSetupHandler(s settings.SystemSetup) *InitialSetupHandler {
	return &InitialSetupHandler{s}
}

func (h *InitialSetupHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if mediaType != "application/x-www-form-urlencoded" {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	subscribe, err := func() (bool, error) {
		v, ok := r.Form["subscribe_newsletter"]

		if !ok {
			return false, nil
		}

		if len(v) != 1 {
			return false, errors.New("Invalid multiple subscribe form values!")
		}

		if v[0] != "on" {
			return false, errors.New("Invalid subscribe form value!")
		}

		return true, nil
	}()

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println("Error parsing subscribe option:", err)
		return
	}

	email, err := func() (string, error) {
		if !subscribe {
			return "", nil
		}

		v, ok := r.Form["email"]

		if !ok {
			return "", errors.New("Invalid Email")
		}

		if len(v) != 1 {
			return "", errors.New("Invalid Email")
		}

		return v[0], nil
	}()

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println("Error getting email address", err)
		return
	}

	mailKind := r.Form.Get("email_kind")

	if err := h.s.SetInitialOptions(r.Context(), settings.InitialSetupOptions{
		SubscribeToNewsletter: subscribe,
		MailKind:              settings.SetupMailKind(mailKind),
		Email:                 email,
	}); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println("Error setting initial options:", err)
		return
	}
}
