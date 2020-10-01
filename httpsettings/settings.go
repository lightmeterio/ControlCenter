package httpsettings

import (
	"errors"
	"fmt"
	"gitlab.com/lightmeter/controlcenter/httpmiddleware"
	"gitlab.com/lightmeter/controlcenter/settings"
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

func (h *Settings) HttpInitialSetup(mux *http.ServeMux) {
	chain := httpmiddleware.New()
	mux.Handle("/settings/initialSetup", chain.WithError(httpmiddleware.CustomHTTPHandler(h.InitialSetupHandler)))
}

func (h *Settings) InitialSetupHandler(w http.ResponseWriter, r *http.Request) error {
	if err := h.handleForm(w, r); err != nil {
		return httpmiddleware.NewHTTPStatusCodeError(http.StatusInternalServerError, err)
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
		return httpmiddleware.NewHTTPStatusCodeError(http.StatusBadRequest, fmt.Errorf("Error parsing subscribe option: %w", err))
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
		return httpmiddleware.NewHTTPStatusCodeError(http.StatusBadRequest, fmt.Errorf("Error getting email address: %w", err))
	}

	mailKind := r.Form.Get("email_kind")

	if err := h.s.SetOptions(r.Context(), settings.InitialOptions{
		SubscribeToNewsletter: subscribe,
		MailKind:              settings.SetupMailKind(mailKind),
		Email:                 email,
	}); err != nil {
		return httpmiddleware.NewHTTPStatusCodeError(http.StatusInternalServerError, fmt.Errorf("Error setting initial options: %w", err))
	}

	return nil
}

func (h *Settings) HttpNotificationSettings(mux *http.ServeMux) {
	chain := httpmiddleware.New()
	mux.Handle("/settings/notificationSettings", chain.WithError(httpmiddleware.CustomHTTPHandler(h.NotificationSettingsHandler)))
}

func (h *Settings) NotificationSettingsHandler(w http.ResponseWriter, r *http.Request) error {
	if err := h.handleForm(w, r); err != nil {
		return httpmiddleware.NewHTTPStatusCodeError(http.StatusInternalServerError, err)
	}

	if len(r.Form) != 3 {
		return httpmiddleware.NewHTTPStatusCodeError(http.StatusInternalServerError, errors.New("Error to many values in form"))
	}

	messengerKind := r.Form.Get("messenger_kind")
	if messengerKind != "slack" {
		return httpmiddleware.NewHTTPStatusCodeError(http.StatusBadRequest, fmt.Errorf("Error messenger kind option is bad %v", messengerKind))
	}

	messengerToken := r.Form.Get("messenger_token")
	if messengerToken == "" {
		return httpmiddleware.NewHTTPStatusCodeError(http.StatusBadRequest, fmt.Errorf("Error messenger token option is bad %v", messengerToken))
	}

	messengerChannel := r.Form.Get("messenger_channel")
	if messengerChannel == "" {
		return httpmiddleware.NewHTTPStatusCodeError(http.StatusBadRequest, fmt.Errorf("Error messenger channel option is bad %v", messengerChannel))
	}

	if err := h.s.SetOptions(r.Context(), settings.SlackNotificationsSettings{
		Kind:        messengerKind,
		BearerToken: messengerToken,
		Channel:     messengerChannel,
	}); err != nil {
		return httpmiddleware.NewHTTPStatusCodeError(http.StatusInternalServerError, fmt.Errorf("Error notification setting options: %w", err))
	}

	return nil
}
