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
	"gitlab.com/lightmeter/controlcenter/pkg/httperror"
	"gitlab.com/lightmeter/controlcenter/po"
	"gitlab.com/lightmeter/controlcenter/settings"
	"gitlab.com/lightmeter/controlcenter/settings/globalsettings"
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
	handlers             map[string]func(http.ResponseWriter, *http.Request) error
}

func NewSettings(writer *meta.AsyncWriter,
	reader *meta.Reader,
	initialSetupSettings *settings.InitialSetupSettings,
	notificationCenter notification.Center,
) *Settings {
	s := &Settings{writer: writer, reader: reader, initialSetupSettings: initialSetupSettings, notificationCenter: notificationCenter}
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
		SlackNotificationSettings struct {
			BearerToken string `json:"bearer_token"`
			Channel     string `json:"channel"`
			Enabled     *bool  `json:"enabled"`
			Language    string `json:"language"`
		} `json:"slack_notifications"`
		GeneralSettings struct {
			PostfixPublicIP net.IP `json:"postfix_public_ip"`
			AppLanguage     string `json:"app_language"`
		} `json:"general"`
	}{}

	ctx := r.Context()

	slackSettings, err := settings.GetSlackNotificationsSettings(ctx, h.reader)

	if err != nil && !errors.Is(err, meta.ErrNoSuchKey) {
		return httperror.NewHTTPStatusCodeError(http.StatusInternalServerError, errorutil.Wrap(err))
	}

	if slackSettings != nil {
		allCurrentSettings.SlackNotificationSettings.BearerToken = slackSettings.BearerToken
		allCurrentSettings.SlackNotificationSettings.Channel = slackSettings.Channel
		allCurrentSettings.SlackNotificationSettings.Enabled = &slackSettings.Enabled
		allCurrentSettings.SlackNotificationSettings.Language = slackSettings.Language
	}

	var globalSettings globalsettings.Settings

	err = h.reader.RetrieveJson(ctx, globalsettings.SettingsKey, &globalSettings)

	if err != nil && !errors.Is(err, meta.ErrNoSuchKey) {
		return httperror.NewHTTPStatusCodeError(http.StatusInternalServerError, errorutil.Wrap(err))
	}

	if err == nil {
		allCurrentSettings.GeneralSettings.PostfixPublicIP = globalSettings.LocalIP
		allCurrentSettings.GeneralSettings.AppLanguage = globalSettings.APPLanguage
	}

	return httputil.WriteJson(w, &allCurrentSettings, http.StatusOK)
}

func (h *Settings) GeneralSettingsHandler(w http.ResponseWriter, r *http.Request) error {
	if err := handleForm(w, r); err != nil {
		return httperror.NewHTTPStatusCodeError(http.StatusInternalServerError, errorutil.Wrap(err))
	}

	localIPRaw := r.Form.Get("postfixPublicIP")
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

	currentSettings := globalsettings.Settings{}

	err := h.reader.RetrieveJson(r.Context(), globalsettings.SettingsKey, &currentSettings)

	if err != nil && !errors.Is(err, meta.ErrNoSuchKey) {
		return httperror.NewHTTPStatusCodeError(http.StatusBadRequest, errorutil.Wrap(err, "Error fetching general configuration"))
	}

	s := globalsettings.Settings{LocalIP: localIP, APPLanguage: appLanguage}

	if err := mergo.Merge(&s, currentSettings); err != nil {
		return httperror.NewHTTPStatusCodeError(http.StatusBadRequest, errorutil.Wrap(err, "Error handling settings"))
	}

	result := h.writer.StoreJson(globalsettings.SettingsKey, &s)

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

	result := h.writer.StoreJson(globalsettings.SettingsKey, &s)

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

func (h *Settings) NotificationSettingsHandler(w http.ResponseWriter, r *http.Request) error {
	if err := handleForm(w, r); err != nil {
		return httperror.NewHTTPStatusCodeError(http.StatusInternalServerError, err)
	}

	messengerKind := r.Form.Get("messenger_kind")
	if messengerKind != "slack" {
		err := errorutil.Wrap(fmt.Errorf("Error messenger kind option is bad %v", messengerKind))
		return httperror.NewHTTPStatusCodeError(http.StatusBadRequest, err)
	}

	messengerToken := r.Form.Get("messenger_token")
	if messengerToken == "" {
		err := errorutil.Wrap(fmt.Errorf("Error messenger token option is bad %v", messengerToken))
		return httperror.NewHTTPStatusCodeError(http.StatusBadRequest, err)
	}

	messengerChannel := r.Form.Get("messenger_channel")
	if messengerChannel == "" {
		err := errorutil.Wrap(fmt.Errorf("Error messenger channel option is bad %v", messengerChannel))
		return httperror.NewHTTPStatusCodeError(http.StatusBadRequest, err)
	}

	messengerEnabled := r.Form.Get("messenger_enabled")
	if messengerEnabled != "false" && messengerEnabled != "true" {
		err := errorutil.Wrap(fmt.Errorf("Error messenger enabled option is bad %v", messengerEnabled))
		return httperror.NewHTTPStatusCodeError(http.StatusBadRequest, err)
	}

	messengerLanguage := r.Form.Get("messenger_language")
	if messengerLanguage == "" {
		err := errorutil.Wrap(fmt.Errorf("Error messenger language option is missing %v", messengerLanguage))
		return httperror.NewHTTPStatusCodeError(http.StatusBadRequest, err)
	}

	if !po.IsLanguageSupported(messengerLanguage) {
		err := errorutil.Wrap(fmt.Errorf("Error messenger language option is bad %v", messengerLanguage))

		return httperror.NewHTTPStatusCodeError(http.StatusBadRequest, err)
	}

	slackNotificationsSettings := settings.SlackNotificationsSettings{
		Kind:        messengerKind,
		BearerToken: messengerToken,
		Channel:     messengerChannel,
		Enabled:     messengerEnabled == "true",
		Language:    messengerLanguage,
	}

	if err := h.notificationCenter.AddSlackNotifier(slackNotificationsSettings); err != nil {
		err := errorutil.Wrap(err, "Error register slack notifier "+err.Error())
		return httperror.NewHTTPStatusCodeError(http.StatusBadRequest, err)
	}

	if err := settings.SetSlackNotificationsSettings(r.Context(), h.writer, slackNotificationsSettings); err != nil {
		err := errorutil.Wrap(err, "Error notification setting options")
		return httperror.NewHTTPStatusCodeError(http.StatusInternalServerError, err)
	}

	return nil
}
