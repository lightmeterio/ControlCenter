package httpsettings

import (
	"errors"
	"fmt"
	"gitlab.com/lightmeter/controlcenter/httpmiddleware"
	"gitlab.com/lightmeter/controlcenter/notification"
	"gitlab.com/lightmeter/controlcenter/settings"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"mime"
	"net/http"
)

type Settings struct {
	s                  settings.SystemSetup
	notificationCenter notification.Center
}

func NewSettings(s settings.SystemSetup, notificationCenter notification.Center) *Settings {
	return &Settings{s, notificationCenter}
}

func (h *Settings) handleForm(w http.ResponseWriter, r *http.Request) error {
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

func (h *Settings) HttpInitialSetup(mux *http.ServeMux) {
	chain := httpmiddleware.WithDefaultTimeout()
	mux.Handle("/settings/initialSetup", chain.WithError(httpmiddleware.CustomHTTPHandler(h.InitialSetupHandler)))
}

func (h *Settings) InitialSetupHandler(w http.ResponseWriter, r *http.Request) error {
	if err := h.handleForm(w, r); err != nil {
		return httpmiddleware.NewHTTPStatusCodeError(http.StatusInternalServerError, errorutil.Wrap(err))
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
		err = errorutil.Wrap(err, "Error parsing subscribe option")
		return httpmiddleware.NewHTTPStatusCodeError(http.StatusBadRequest, err)
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
		err = errorutil.Wrap(err, "Error getting email address")
		return httpmiddleware.NewHTTPStatusCodeError(http.StatusBadRequest, err)
	}

	mailKind := r.Form.Get("email_kind")

	if err := h.s.SetOptions(r.Context(), settings.InitialOptions{
		SubscribeToNewsletter: subscribe,
		MailKind:              settings.SetupMailKind(mailKind),
		Email:                 email,
	}); err != nil {
		err = errorutil.Wrap(err, "Error setting initial options")
		return httpmiddleware.NewHTTPStatusCodeError(http.StatusInternalServerError, err)
	}

	return nil
}

func (h *Settings) HttpNotificationSettings(mux *http.ServeMux) {
	chain := httpmiddleware.WithDefaultTimeout()
	mux.Handle("/settings/notificationSettings", chain.WithError(httpmiddleware.CustomHTTPHandler(h.NotificationSettingsHandler)))
}

func (h *Settings) NotificationSettingsHandler(w http.ResponseWriter, r *http.Request) error {
	if err := h.handleForm(w, r); err != nil {
		return httpmiddleware.NewHTTPStatusCodeError(http.StatusInternalServerError, err)
	}

	if len(r.Form) != 3 {
		err := errorutil.Wrap(errors.New("Error to many values in form"))
		return httpmiddleware.NewHTTPStatusCodeError(http.StatusInternalServerError, err)
	}

	messengerKind := r.Form.Get("messenger_kind")
	if messengerKind != "slack" {
		err := errorutil.Wrap(fmt.Errorf("Error messenger kind option is bad %v", messengerKind))
		return httpmiddleware.NewHTTPStatusCodeError(http.StatusBadRequest, err)
	}

	messengerToken := r.Form.Get("messenger_token")
	if messengerToken == "" {
		err := errorutil.Wrap(fmt.Errorf("Error messenger token option is bad %v", messengerToken))
		return httpmiddleware.NewHTTPStatusCodeError(http.StatusBadRequest, err)
	}

	messengerChannel := r.Form.Get("messenger_channel")
	if messengerChannel == "" {
		err := errorutil.Wrap(fmt.Errorf("Error messenger channel option is bad %v", messengerChannel))
		return httpmiddleware.NewHTTPStatusCodeError(http.StatusBadRequest, err)
	}

	slackNotificationsSettings := settings.SlackNotificationsSettings{
		Kind:        messengerKind,
		BearerToken: messengerToken,
		Channel:     messengerChannel,
	}

	if err := h.s.SetOptions(r.Context(), slackNotificationsSettings); err != nil {
		err := errorutil.Wrap(err, "Error notification setting options")
		return httpmiddleware.NewHTTPStatusCodeError(http.StatusInternalServerError, err)
	}

	if err := h.notificationCenter.AddSlackNotifier(slackNotificationsSettings); err != nil {
		err := errorutil.Wrap(err, "Error register slack notifier ")
		return httpmiddleware.NewHTTPStatusCodeError(http.StatusInternalServerError, err)
	}

	return nil
}
