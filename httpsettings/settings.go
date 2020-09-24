package httpsettings

import (
	"errors"
	"fmt"
	"gitlab.com/lightmeter/controlcenter/settings"
	"log"
	"mime"
	"net/http"
)

type Settings struct {
	s settings.SystemSetup
}

func NewSettings(s settings.SystemSetup) *Settings {
	return &Settings{s}
}

func (h *Settings) handleForm(w http.ResponseWriter, r *http.Request) error {
	if r.Method != "POST" {
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

func (h *Settings) InitialSetupHandler(w http.ResponseWriter, r *http.Request) {
	if err := h.handleForm(w, r); err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	subscribe, err := func() (bool, error) {
		v, ok := r.Form["subscribe_newsletter"]

		if !ok {
			return false, nil
		}

		if len(v) != 1 {
			return false, fmt.Errorf("Invalid multiple subscribe form values, count:%v", len(v))
		}

		if v[0] != "on" {
			return false, fmt.Errorf("Invalid subscribe form value!, value: %v", v[0])
		}

		return true, nil
	}()

	if err != nil {
		log.Println("Error parsing subscribe option:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	email, err := func() (string, error) {
		if !subscribe {
			return "", nil
		}

		v, ok := r.Form["email"]

		errFormValidationInvalidEmail := errors.New("Invalid Email")

		if !ok {
			return "", errFormValidationInvalidEmail
		}

		if len(v) != 1 {
			return "", errFormValidationInvalidEmail
		}

		return v[0], nil
	}()

	if err != nil {
		log.Println("Error getting email address:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	mailKind := r.Form.Get("email_kind")

	if err := h.s.SetOptions(r.Context(), settings.InitialOptions{
		SubscribeToNewsletter: subscribe,
		MailKind:              settings.SetupMailKind(mailKind),
		Email:                 email,
	}); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println("Error setting initial options:", err)
		return
	}
}

func (h *Settings) NotificationSettingsHandler(w http.ResponseWriter, r *http.Request) {
	if err := h.handleForm(w, r); err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if len(r.Form) != 3 {
		log.Println("Error to many values in form")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	messengerKind := r.Form.Get("messenger_kind")
	if messengerKind != "slack" {
		log.Println("Error messenger kind option is bad ", messengerKind)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	messengerToken := r.Form.Get("messenger_token")
	if messengerToken == "" {
		log.Println("Error messenger token option is bad ", messengerToken)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	messengerChannel := r.Form.Get("messenger_channel")
	if messengerChannel == "" {
		log.Println("Error messenger channel option is bad ", messengerChannel)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := h.s.SetOptions(r.Context(), settings.SlackNotificationsSettings{
		Kind:        messengerKind,
		BearerToken: messengerToken,
		Channel:     messengerChannel,
	}); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println("Error notification setting options:", err)
		return
	}
}
