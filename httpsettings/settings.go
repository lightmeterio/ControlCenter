package httpsettings

import (
	"errors"
	"fmt"
	"gitlab.com/lightmeter/controlcenter/httpmiddleware"
	"gitlab.com/lightmeter/controlcenter/localrbl"
	"gitlab.com/lightmeter/controlcenter/meta"
	"gitlab.com/lightmeter/controlcenter/notification"
	"gitlab.com/lightmeter/controlcenter/settings"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/httputil"
	"mime"
	"net"
	"net/http"
)

type Settings struct {
	writer *meta.AsyncWriter
	reader *meta.Reader

	initialSetupSettings *settings.InitialSetupSettings
	notificationCenter   notification.Center
}

func NewSettings(writer *meta.AsyncWriter,
	reader *meta.Reader,
	initialSetupSettings *settings.InitialSetupSettings,
	notificationCenter notification.Center,
) *Settings {
	return &Settings{writer, reader, initialSetupSettings, notificationCenter}
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

func (h *Settings) SetupMux(mux *http.ServeMux) {
	chain := httpmiddleware.WithDefaultTimeout()

	mux.Handle("/settings", chain.WithError(httpmiddleware.CustomHTTPHandler(h.SettingsHandler)))
	mux.Handle("/settings/initialSetup", chain.WithError(httpmiddleware.CustomHTTPHandler(h.InitialSetupHandler)))
	mux.Handle("/settings/notificationSettings", chain.WithError(httpmiddleware.CustomHTTPHandler(h.NotificationSettingsHandler)))
	mux.Handle("/settings/generalSettings", chain.WithError(httpmiddleware.CustomHTTPHandler(h.GeneralSettingsHandler)))
}

func (h *Settings) SettingsHandler(w http.ResponseWriter, r *http.Request) error {
	// For now we only allow fetching settings
	// TODO: use this endpoint as a generic way to set settings, making the other specialized endpoints obsolete.
	// so that /settings?setting=initialSetup does the job of /settings/initialSetup, and so on...
	// TODO: make this endpoint part of the API, on /api/v0/settings
	if r.Method != http.MethodGet {
		return httpmiddleware.NewHTTPStatusCodeError(http.StatusMethodNotAllowed, fmt.Errorf("Error http method mismatch: %v", r.Method))
	}

	// TODO: this structure should somehow be dynamic and easily extensible for future new settings we add,
	// also supporting optional settings
	allCurrentSettings := struct {
		SlackNotificationSettings struct {
			BearerToken string `json:"bearer_token"`
			Channel     string `json:"channel"`
			Enabled     *bool  `json:"enabled"`
		} `json:"slack_notifications"`
		GeneralSettings struct {
			PostfixPublicIP net.IP `json:"postfix_public_ip"`
		} `json:"general"`
	}{}

	ctx := r.Context()

	slackSettings, err := settings.GetSlackNotificationsSettings(ctx, h.reader)

	if err != nil && !errors.Is(err, meta.ErrNoSuchKey) {
		return httpmiddleware.NewHTTPStatusCodeError(http.StatusInternalServerError, errorutil.Wrap(err))
	}

	if slackSettings != nil {
		allCurrentSettings.SlackNotificationSettings.BearerToken = slackSettings.BearerToken
		allCurrentSettings.SlackNotificationSettings.Channel = slackSettings.Channel
		allCurrentSettings.SlackNotificationSettings.Enabled = &slackSettings.Enabled
	}

	var localRBLSettings localrbl.Settings

	err = h.reader.RetrieveJson(ctx, localrbl.SettingsKey, &localRBLSettings)

	if err != nil && !errors.Is(err, meta.ErrNoSuchKey) {
		return httpmiddleware.NewHTTPStatusCodeError(http.StatusInternalServerError, errorutil.Wrap(err))
	}

	if err == nil {
		allCurrentSettings.GeneralSettings.PostfixPublicIP = localRBLSettings.LocalIP
	}

	return httputil.WriteJson(w, &allCurrentSettings, http.StatusOK)
}

func (h *Settings) GeneralSettingsHandler(w http.ResponseWriter, r *http.Request) error {
	if err := handleForm(w, r); err != nil {
		return httpmiddleware.NewHTTPStatusCodeError(http.StatusInternalServerError, errorutil.Wrap(err))
	}

	localIP := net.ParseIP(r.Form.Get("postfixPublicIP"))

	if localIP == nil {
		return httpmiddleware.NewHTTPStatusCodeError(http.StatusBadRequest, fmt.Errorf("Invalid IP address"))
	}

	s := localrbl.Settings{LocalIP: localIP}

	result := h.writer.StoreJson(localrbl.SettingsKey, &s)

	select {
	case err := <-result.Done():
		if err != nil {
			return httpmiddleware.NewHTTPStatusCodeError(http.StatusInternalServerError, errorutil.Wrap(err))
		}

		return nil
	case <-r.Context().Done():
		return httpmiddleware.NewHTTPStatusCodeError(http.StatusInternalServerError, fmt.Errorf("Failed to store local rbl settings"))
	}
}

func (h *Settings) InitialSetupHandler(w http.ResponseWriter, r *http.Request) error {
	if err := handleForm(w, r); err != nil {
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

	err = h.initialSetupSettings.Set(r.Context(), h.writer, settings.InitialOptions{
		SubscribeToNewsletter: subscribe,
		MailKind:              settings.SetupMailKind(mailKind),
		Email:                 email,
	})

	if err != nil {
		err = errorutil.Wrap(err, "Error setting initial options")
		return httpmiddleware.NewHTTPStatusCodeError(http.StatusInternalServerError, err)
	}

	return nil
}

func (h *Settings) NotificationSettingsHandler(w http.ResponseWriter, r *http.Request) error {
	if err := handleForm(w, r); err != nil {
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

	messengerEnabled := r.Form.Get("messenger_enabled")
	if messengerEnabled != "false" && messengerEnabled != "true" {
		err := errorutil.Wrap(fmt.Errorf("Error messenger enabled option is bad %v", messengerEnabled))
		return httpmiddleware.NewHTTPStatusCodeError(http.StatusBadRequest, err)
	}

	slackNotificationsSettings := settings.SlackNotificationsSettings{
		Kind:        messengerKind,
		BearerToken: messengerToken,
		Channel:     messengerChannel,
		Enabled:     messengerEnabled == "true",
	}

	if err := h.notificationCenter.AddSlackNotifier(slackNotificationsSettings); err != nil {
		err := errorutil.Wrap(err, "Error register slack notifier "+err.Error())
		return httpmiddleware.NewHTTPStatusCodeError(http.StatusBadRequest, err)
	}

	if err := settings.SetSlackNotificationsSettings(r.Context(), h.writer, slackNotificationsSettings); err != nil {
		err := errorutil.Wrap(err, "Error notification setting options")
		return httpmiddleware.NewHTTPStatusCodeError(http.StatusInternalServerError, err)
	}

	return nil
}

func (h *Settings) HttpSettingsPage(mux *http.ServeMux) {
	mux.Handle("/settingspage", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/settingspage.i18n.html", http.StatusSeeOther)
	}))
}
