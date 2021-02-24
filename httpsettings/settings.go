// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package httpsettings

import (
	"errors"
	"fmt"
	"github.com/imdario/mergo"
	"gitlab.com/lightmeter/controlcenter/httpauth/auth"
	"gitlab.com/lightmeter/controlcenter/httpmiddleware"
	"gitlab.com/lightmeter/controlcenter/meta"
	"gitlab.com/lightmeter/controlcenter/notification"
	"gitlab.com/lightmeter/controlcenter/notification/email"
	"gitlab.com/lightmeter/controlcenter/notification/slack"
	"gitlab.com/lightmeter/controlcenter/pkg/httperror"
	"gitlab.com/lightmeter/controlcenter/po"
	"gitlab.com/lightmeter/controlcenter/settings"
	"gitlab.com/lightmeter/controlcenter/settings/globalsettings"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/httputil"
	"mime"
	"net"
	"net/http"
	"net/url"
	"strconv"
)

type Settings struct {
	writer *meta.AsyncWriter
	reader *meta.Reader

	initialSetupSettings *settings.InitialSetupSettings
	notificationCenter   *notification.Center
	handlers             map[string]func(http.ResponseWriter, *http.Request) error
}

func NewSettings(writer *meta.AsyncWriter,
	reader *meta.Reader,
	initialSetupSettings *settings.InitialSetupSettings,
	notificationCenter *notification.Center,
) *Settings {
	s := &Settings{
		writer:               writer,
		reader:               reader,
		initialSetupSettings: initialSetupSettings,
		notificationCenter:   notificationCenter,
	}
	s.handlers = map[string]func(http.ResponseWriter, *http.Request) error{
		"initSetup":    s.InitialSetupHandler,
		"notification": s.NotificationSettingsHandler,
		"general":      s.GeneralSettingsHandler,
	}

	return s
}

func handleForm(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodPost {
		return fmt.Errorf("Error http method mismatch: %v", r.Method)
	}

	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		return fmt.Errorf("Error parse media type: %w", err)
	}

	if mediaType != "application/x-www-form-urlencoded" {
		return fmt.Errorf("Error media type mismatch: %w", err)
	}

	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("Error parse form: %w", err)
	}

	return nil
}

func (h *Settings) HttpSetup(mux *http.ServeMux, auth *auth.Authenticator) {
	chain := httpmiddleware.WithDefaultStack(auth)

	mux.Handle("/settings", chain.WithError(httpmiddleware.CustomHTTPHandler(h.SettingsForward)))
}

func (h *Settings) SettingsForward(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		return httperror.NewHTTPStatusCodeError(http.StatusMethodNotAllowed, fmt.Errorf("Error http method mismatch: %v", r.Method))
	}

	if r.Method == http.MethodGet {
		return h.SettingsHandler(w, r)
	}

	kind := r.URL.Query().Get("setting")
	if kind == "" {
		return httperror.NewHTTPStatusCodeError(http.StatusBadRequest, errors.New("Error query parameter setting is missing"))
	}

	handler, ok := h.handlers[kind]
	if !ok {
		return httperror.NewHTTPStatusCodeError(http.StatusBadRequest, errors.New("Error handler type is not supported"))
	}

	return handler(w, r)
}

func (h *Settings) SettingsHandler(w http.ResponseWriter, r *http.Request) error {
	// For now we only allow fetching settings
	// TODO: use this endpoint as a generic way to set settings, making the other specialized endpoints obsolete.
	// so that /settings?setting=initialSetup does the job of /settings/initialSetup, and so on...
	// TODO: make this endpoint part of the API, on /api/v0/settings
	if r.Method != http.MethodGet {
		return httperror.NewHTTPStatusCodeError(http.StatusMethodNotAllowed, fmt.Errorf("Error http method mismatch: %v", r.Method))
	}

	// TODO: this structure should somehow be dynamic and easily extensible for future new settings we add,
	// also supporting optional settings
	allCurrentSettings := struct {
		SlackNotification slack.Settings          `json:"slack_notifications"`
		EmailNotification email.Settings          `json:"email_notifications"`
		Notification      notification.Settings   `json:"notifications"`
		General           globalsettings.Settings `json:"general"`
	}{}

	ctx := r.Context()

	slackSettings, err := slack.GetSettings(ctx, h.reader)
	if err != nil && !errors.Is(err, meta.ErrNoSuchKey) {
		return httperror.NewHTTPStatusCodeError(http.StatusInternalServerError, errorutil.Wrap(err))
	}

	emailSettings, err := email.GetSettings(ctx, h.reader)
	if err != nil && !errors.Is(err, meta.ErrNoSuchKey) {
		return httperror.NewHTTPStatusCodeError(http.StatusInternalServerError, errorutil.Wrap(err))
	}

	notificationSettings, err := notification.GetSettings(ctx, h.reader)
	if err != nil && !errors.Is(err, meta.ErrNoSuchKey) {
		return httperror.NewHTTPStatusCodeError(http.StatusInternalServerError, errorutil.Wrap(err))
	}

	globalSettings, err := globalsettings.GetSettings(ctx, h.reader)
	if err != nil && !errors.Is(err, meta.ErrNoSuchKey) {
		return httperror.NewHTTPStatusCodeError(http.StatusInternalServerError, errorutil.Wrap(err))
	}

	if slackSettings != nil {
		allCurrentSettings.SlackNotification = *slackSettings
	}

	if notificationSettings != nil {
		allCurrentSettings.Notification = *notificationSettings
	}

	if emailSettings != nil {
		allCurrentSettings.EmailNotification = *emailSettings
	}

	if globalSettings != nil {
		allCurrentSettings.General = *globalSettings
	}

	return httputil.WriteJson(w, &allCurrentSettings, http.StatusOK)
}

func (h *Settings) GeneralSettingsHandler(w http.ResponseWriter, r *http.Request) error {
	if err := handleForm(w, r); err != nil {
		return httperror.NewHTTPStatusCodeError(http.StatusInternalServerError, errorutil.Wrap(err))
	}

	localIPRaw := r.Form.Get("postfix_public_ip")
	appLanguage := r.Form.Get("app_language")

	if appLanguage == "" && localIPRaw == "" {
		err := errorutil.Wrap(errors.New("values are missing"))
		return httperror.NewHTTPStatusCodeError(http.StatusBadRequest, err)
	}

	var localIP net.IP
	if localIPRaw != "" {
		localIP = net.ParseIP(localIPRaw)
		if localIP == nil {
			return httperror.NewHTTPStatusCodeError(http.StatusBadRequest, fmt.Errorf("Invalid IP address"))
		}
	}

	if appLanguage != "" && !po.IsLanguageSupported(appLanguage) {
		err := errorutil.Wrap(fmt.Errorf("Error app language option is bad %v", appLanguage))
		return httperror.NewHTTPStatusCodeError(http.StatusBadRequest, err)
	}

	publicURL := r.Form.Get("public_url")

	currentSettings, err := func() (globalsettings.Settings, error) {
		s, err := globalsettings.GetSettings(r.Context(), h.reader)

		if err != nil {
			return globalsettings.Settings{}, errorutil.Wrap(err)
		}

		return *s, nil
	}()

	if err != nil && !errors.Is(err, meta.ErrNoSuchKey) {
		return httperror.NewHTTPStatusCodeError(http.StatusBadRequest, errorutil.Wrap(err, "Error fetching general configuration"))
	}

	s := globalsettings.Settings{LocalIP: localIP, APPLanguage: appLanguage, PublicURL: publicURL}

	if err := mergo.Merge(&s, currentSettings); err != nil {
		return httperror.NewHTTPStatusCodeError(http.StatusInternalServerError, errorutil.Wrap(err, "Error handling settings"))
	}

	if err := globalsettings.SetSettings(r.Context(), h.writer, s); err != nil {
		return httperror.NewHTTPStatusCodeError(http.StatusInternalServerError, errorutil.Wrap(err))
	}

	return nil
}

func (h *Settings) InitialSetupHandler(w http.ResponseWriter, r *http.Request) error {
	if err := handleForm(w, r); err != nil {
		return httperror.NewHTTPStatusCodeError(http.StatusInternalServerError, errorutil.Wrap(err))
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
		return httperror.NewHTTPStatusCodeError(http.StatusBadRequest, err)
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
		return httperror.NewHTTPStatusCodeError(http.StatusBadRequest, err)
	}

	mailKind := r.Form.Get("email_kind")

	err = h.initialSetupSettings.Set(r.Context(), h.writer, settings.InitialOptions{
		SubscribeToNewsletter: subscribe,
		MailKind:              settings.SetupMailKind(mailKind),
		Email:                 email,
	})

	if err != nil {
		err = errorutil.Wrap(err, "Error setting initial options")
		return httperror.NewHTTPStatusCodeError(http.StatusInternalServerError, err)
	}

	appLanguage := r.Form.Get("app_language")
	if appLanguage != "" && !po.IsLanguageSupported(appLanguage) {
		err := errorutil.Wrap(fmt.Errorf("Error app language option is bad %v", appLanguage))

		return httperror.NewHTTPStatusCodeError(http.StatusBadRequest, err)
	}

	s := globalsettings.Settings{APPLanguage: appLanguage}

	postfixPublicIp := r.Form.Get("postfix_public_ip")
	if postfixPublicIp != "" {
		var localIP net.IP
		if postfixPublicIp != "" {
			localIP = net.ParseIP(postfixPublicIp)
			if localIP == nil {
				return httperror.NewHTTPStatusCodeError(http.StatusBadRequest, fmt.Errorf("Invalid IP address"))
			}
		}

		s = globalsettings.Settings{APPLanguage: appLanguage, LocalIP: localIP}
	}

	result := h.writer.StoreJson(globalsettings.SettingKey, &s)

	select {
	case err := <-result.Done():
		if err != nil {
			return httperror.NewHTTPStatusCodeError(http.StatusInternalServerError, errorutil.Wrap(err))
		}

		return nil
	case <-r.Context().Done():
		return httperror.NewHTTPStatusCodeError(http.StatusInternalServerError, fmt.Errorf("Failed to store global settings"))
	}
}

func buildEmailSettingsFromForm(form url.Values) (email.Settings, bool, error) {
	serverName := form.Get("email_notification_server_name")
	username := form.Get("email_notification_username")
	password := form.Get("email_notification_password")
	sender := form.Get("email_notification_sender")
	recipients := form.Get("email_notification_recipients")

	securityStr := form.Get("email_notification_security_type")
	authStr := form.Get("email_notification_auth_method")
	portStr := form.Get("email_notification_port")

	argsAreEmpty := func(s ...string) bool {
		for _, s := range s {
			if len(s) != 0 {
				return false
			}
		}

		return true
	}

	if argsAreEmpty(serverName, username, password, sender, recipients, securityStr, authStr, portStr) {
		return email.Settings{}, false, nil
	}

	security, err := email.ParseSecurityType(securityStr)
	if err != nil {
		return email.Settings{}, false, errorutil.Wrap(err)
	}

	auth, err := email.ParseAuthMethod(authStr)
	if err != nil {
		return email.Settings{}, false, errorutil.Wrap(err)
	}

	if (portStr == "0" || portStr == "") && auth == email.AuthMethodNone &&
		security == email.SecurityTypeNone &&
		argsAreEmpty(serverName, username, password, sender, recipients) {
		return email.Settings{}, false, nil
	}

	port, err := strconv.ParseInt(portStr, 10, 16)
	if err != nil {
		return email.Settings{}, false, errorutil.Wrap(err)
	}

	enabled, err := strconv.ParseBool(form.Get("email_notification_enabled"))
	if err != nil {
		return email.Settings{}, false, errorutil.Wrap(err)
	}

	return email.Settings{
		Enabled:      enabled,
		Sender:       sender,
		Recipients:   recipients,
		ServerName:   serverName,
		ServerPort:   int(port),
		SecurityType: security,
		AuthMethod:   auth,
		Username:     username,
		Password:     password,
	}, true, nil
}

func buildSlackSettingsFromForm(form url.Values) (slack.Settings, bool, error) {
	if len(form.Get("messenger_channel")) == 0 && len(form.Get("messenger_token")) == 0 {
		return slack.Settings{}, false, nil
	}

	messengerToken := form.Get("messenger_token")
	if messengerToken == "" {
		return slack.Settings{}, false, errorutil.Wrap(fmt.Errorf("Error messenger token option is bad %v", messengerToken))
	}

	messengerChannel := form.Get("messenger_channel")
	if messengerChannel == "" {
		return slack.Settings{}, false, errorutil.Wrap(fmt.Errorf("Error messenger channel option is bad %v", messengerChannel))
	}

	messengerEnabled, err := strconv.ParseBool(form.Get("messenger_enabled"))
	if err != nil {
		return slack.Settings{}, false, errorutil.Wrap(err)
	}

	return slack.Settings{
		BearerToken: messengerToken,
		Channel:     messengerChannel,
		Enabled:     messengerEnabled,
	}, true, nil
}

func buildNotificationSettingsFromForm(form url.Values) (notification.Settings, bool, error) {
	language := form.Get("notification_language")

	if language == "" {
		return notification.Settings{}, false, nil
	}

	// TODO: move this check to the i18n package!
	if !po.IsLanguageSupported(language) {
		return notification.Settings{}, false, errorutil.Wrap(fmt.Errorf("Invalid language: %v", language))
	}

	return notification.Settings{
		Language: language,
	}, true, nil
}

func (h *Settings) NotificationSettingsHandler(w http.ResponseWriter, r *http.Request) error {
	if err := handleForm(w, r); err != nil {
		return httperror.NewHTTPStatusCodeError(http.StatusInternalServerError, err)
	}

	slackSettings, shouldSetSlack, err := buildSlackSettingsFromForm(r.Form)
	if err != nil {
		return httperror.NewHTTPStatusCodeError(http.StatusBadRequest, errorutil.Wrap(err))
	}

	mailSettings, shouldSetEmail, err := buildEmailSettingsFromForm(r.Form)
	if err != nil {
		return httperror.NewHTTPStatusCodeError(http.StatusBadRequest, errorutil.Wrap(err))
	}

	notificationSettings, shouldSetNotification, err := buildNotificationSettingsFromForm(r.Form)
	if err != nil {
		return httperror.NewHTTPStatusCodeError(http.StatusBadRequest, errorutil.Wrap(err))
	}

	if shouldSetSlack {
		slackNotifier, err := h.notificationCenter.Notifier(slack.SettingKey)
		if err != nil {
			return httperror.NewHTTPStatusCodeError(http.StatusInternalServerError, errorutil.Wrap(err))
		}

		if err := slackNotifier.ValidateSettings(slackSettings); err != nil {
			err := errorutil.Wrap(err, "Error register slack notifier "+err.Error())
			return httperror.NewHTTPStatusCodeError(http.StatusBadRequest, err)
		}

		if err := slack.SetSettings(r.Context(), h.writer, slackSettings); err != nil {
			err := errorutil.Wrap(err, "Error notification setting options")
			return httperror.NewHTTPStatusCodeError(http.StatusInternalServerError, err)
		}
	}

	if shouldSetEmail {
		emailNotifier, err := h.notificationCenter.Notifier(email.SettingKey)
		if err != nil {
			return httperror.NewHTTPStatusCodeError(http.StatusInternalServerError, errorutil.Wrap(err))
		}

		if err := emailNotifier.ValidateSettings(mailSettings); err != nil {
			return httperror.NewHTTPStatusCodeError(http.StatusBadRequest, err)
		}

		if err := email.SetSettings(r.Context(), h.writer, mailSettings); err != nil {
			return httperror.NewHTTPStatusCodeError(http.StatusInternalServerError, errorutil.Wrap(err))
		}
	}

	if shouldSetNotification {
		if err := notification.SetSettings(r.Context(), h.writer, notificationSettings); err != nil {
			return httperror.NewHTTPStatusCodeError(http.StatusInternalServerError, errorutil.Wrap(err))
		}
	}

	return nil
}
