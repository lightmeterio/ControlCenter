// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

//go:generate go run ./templates/gen_template.go

package email

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/mail"
	"strconv"
	"strings"
	"text/template"
	"time"

	sasl "github.com/emersion/go-sasl"
	smtp "github.com/emersion/go-smtp"
	"gitlab.com/lightmeter/controlcenter/i18n/translator"
	"gitlab.com/lightmeter/controlcenter/metadata"
	"gitlab.com/lightmeter/controlcenter/notification/core"
	"gitlab.com/lightmeter/controlcenter/settings/globalsettings"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/stringutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"gitlab.com/lightmeter/controlcenter/version"
)

// TODO: email template (translatable), custom certificate

// this message template is used only by the tests
var messageTemplate = `
Title: {{.Title}}
Description: {{.Description}}
Category: {{.Category}}
Priority: {{.Priority}}
PriorityColor: {{.PriorityColor}}
DetailsURL: {{.DetailsURL}}
PreferencesURL: {{.PreferencesURL}}
PublicURL: {{.PublicURL}}
Version: {{appVersion}}
`

const SettingKey = "messenger_email"

type SecurityType int

const none = "none"

func (t SecurityType) String() string {
	switch t {
	case SecurityTypeNone:
		return none
	case SecurityTypeTLS:
		return "TLS"
	case SecurityTypeSTARTTLS:
		return "STARTTLS"
	default:
		panic("invalid security type")
	}
}

func (t *SecurityType) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.String())
}

var ErrParsingSecurityType = errors.New(`Invalid security type`)

func ParseSecurityType(s string) (SecurityType, error) {
	switch s {
	case none:
		return SecurityTypeNone, nil
	case "STARTTLS":
		return SecurityTypeSTARTTLS, nil
	case "TLS":
		return SecurityTypeTLS, nil
	default:
		return 0, ErrParsingSecurityType
	}
}

func (t *SecurityType) MergoFromString(s string) error {
	v, err := ParseSecurityType(s)
	if err != nil {
		return errorutil.Wrap(err)
	}

	*t = v

	return nil
}

const (
	SecurityTypeNone     SecurityType = 0
	SecurityTypeSTARTTLS SecurityType = 1
	SecurityTypeTLS      SecurityType = 2
)

type AuthMethod int

func (m AuthMethod) String() string {
	switch m {
	case AuthMethodNone:
		return none
	case AuthMethodPassword:
		return "password"
	default:
		panic("invalid auth method")
	}
}

func (m *AuthMethod) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.String())
}

var ErrParsingAuthMethod = errors.New(`Invalid auth method`)

func ParseAuthMethod(s string) (AuthMethod, error) {
	switch s {
	case none:
		return AuthMethodNone, nil
	case "password":
		return AuthMethodPassword, nil
	default:
		return 0, ErrParsingAuthMethod
	}
}

func (m *AuthMethod) MergoFromString(s string) error {
	v, err := ParseAuthMethod(s)
	if err != nil {
		return errorutil.Wrap(err)
	}

	*m = v

	return nil
}

const (
	AuthMethodNone     AuthMethod = 0
	AuthMethodPassword AuthMethod = 1
)

type ServerPort int

func (p *ServerPort) MergoFromString(s string) error {
	v, err := strconv.Atoi(s)
	if err != nil {
		return errorutil.Wrap(err)
	}

	*p = ServerPort(v)

	return nil
}

type Settings struct {
	Enabled bool `json:"enabled"`

	SkipCertCheck bool `json:"skip_cert_check"`

	Sender     string `json:"sender"`
	Recipients string `json:"recipients"`

	ServerName string     `json:"server_name"`
	ServerPort ServerPort `json:"server_port"`

	SecurityType SecurityType `json:"security_type"`
	AuthMethod   AuthMethod   `json:"auth_method"`

	Username stringutil.Sensitive `json:"username,omitempty"`
	Password stringutil.Sensitive `json:"password,omitempty"`
}

func addrFromSettings(s Settings) string {
	return fmt.Sprintf("%s:%d", s.ServerName, s.ServerPort)
}

type SettingsFetcher func() (*Settings, *globalsettings.Settings, error)

type Notifier struct {
	policy          core.Policy
	settingsFetcher SettingsFetcher
	clock           timeutil.Clock
}

func buildTLSConfigFromSettings(settings Settings) *tls.Config {
	if settings.SkipCertCheck {
		//nolint:gosec
		return &tls.Config{InsecureSkipVerify: true}
	}

	return nil
}

func ValidateSettings(settings Settings) (err error) {
	if err := sendOnClient(settings, func(*smtp.Client) error {
		return nil
	}); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func sendOnClient(settings Settings, actionOnClient func(*smtp.Client) error) (err error) {
	addr := addrFromSettings(settings)

	c, err := func() (*smtp.Client, error) {
		if settings.SecurityType != SecurityTypeTLS {
			c, err := smtp.Dial(addr)
			if err != nil {
				return nil, errorutil.Wrap(err)
			}

			return c, nil
		}

		c, err := smtp.DialTLS(addr, buildTLSConfigFromSettings(settings))

		if err != nil {
			return nil, errorutil.Wrap(err)
		}

		return c, nil
	}()

	if err != nil {
		return errorutil.Wrap(err)
	}

	// this call might fail, and that's fine, as the connection
	// normally closes on Quit()
	defer c.Close()

	if ok, _ := c.Extension("STARTTLS"); ok || settings.SecurityType == SecurityTypeSTARTTLS {
		if err := c.StartTLS(buildTLSConfigFromSettings(settings)); err != nil {
			return errorutil.Wrap(err)
		}
	}

	if hasAuth, _ := c.Extension("AUTH"); hasAuth || settings.AuthMethod == AuthMethodPassword {
		// TODO: maybe support OAUTHBEARER (OAuth2) as well?
		auth := sasl.NewPlainClient("", *settings.Username, *settings.Password)

		if err = c.Auth(auth); err != nil {
			return errorutil.Wrap(err)
		}
	}

	if err := actionOnClient(c); err != nil {
		return errorutil.Wrap(err)
	}

	if err := c.Quit(); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func newWithCustomSettingsFetcherAndClock(policy core.Policy, settingsFetcher SettingsFetcher, clock timeutil.Clock) *Notifier {
	return &Notifier{
		policy:          policy,
		settingsFetcher: settingsFetcher,
		clock:           clock,
	}
}

func NewWithCustomSettingsFetcher(policy core.Policy, settingsFetcher SettingsFetcher) *Notifier {
	return newWithCustomSettingsFetcherAndClock(policy, settingsFetcher, &timeutil.RealClock{})
}

type disabledFromSettingsPolicy struct {
	settingsFetcher SettingsFetcher
}

func (p *disabledFromSettingsPolicy) Reject(core.Notification) (bool, error) {
	s, _, err := p.settingsFetcher()

	if err != nil && errors.Is(err, metadata.ErrNoSuchKey) {
		return true, nil
	}

	if err != nil {
		return true, errorutil.Wrap(err)
	}

	return !s.Enabled, nil
}

// FIXME: this function is copied from notification/slack!!!
func New(policy core.Policy, reader metadata.Reader) *Notifier {
	fetcher := func() (*Settings, *globalsettings.Settings, error) {
		settings, err := GetSettings(context.Background(), reader)
		if err != nil {
			return nil, nil, errorutil.Wrap(err)
		}

		globalSettings, err := globalsettings.GetSettings(context.Background(), reader)
		if err != nil {
			return nil, nil, errorutil.Wrap(err)
		}

		return settings, globalSettings, nil
	}

	policies := core.Policies{policy, &disabledFromSettingsPolicy{settingsFetcher: fetcher}}

	return NewWithCustomSettingsFetcher(policies, fetcher)
}

var ErrInvalidEmail = errors.New(`Invalid mail value`)

type templateValues struct {
	Title          string
	Description    string
	Category       string
	Priority       string
	PriorityColor  string
	PublicURL      string
	DetailsURL     string
	PreferencesURL string
}

// as copied from insights.vue CSS code
var colorMap = map[string]string{
	"bad":     "rgb(255, 92, 111)",
	"ok":      "rgb(255, 220, 0)",
	"good":    "rgb(135, 197, 40)",
	"unrated": "#f0f8fc",
}

func priorityToColor(p string) string {
	if c, ok := colorMap[p]; ok {
		return c
	}

	return colorMap["unrated"]
}

func buildTemplateValues(id int64, message core.Message, globalSettings *globalsettings.Settings) templateValues {
	detailsURL := fmt.Sprintf("%s#/insight-card/%v", globalSettings.PublicURL, id)
	preferencesURL := fmt.Sprintf("%s#/settings", globalSettings.PublicURL)

	t := templateValues{
		Title:          message.Title,
		Description:    message.Description,
		PublicURL:      globalSettings.PublicURL,
		DetailsURL:     detailsURL,
		PreferencesURL: preferencesURL,
	}

	// fill the blanks
	for k, v := range message.Metadata {
		switch k {
		case "category":
			t.Category = v
		case "priority":
			t.Priority = v
			t.PriorityColor = priorityToColor(t.Priority)
		}
	}

	return t
}

func buildMessageProperties(translator translator.Translator,
	n core.Notification,
	clock timeutil.Clock,
	settings *Settings,
	globalSettings *globalsettings.Settings,
) (io.Reader, []string, error) {
	template, err := template.New("root").Funcs(template.FuncMap{
		"appVersion": func() string { return version.Version },
		"translate":  translator.Translate,
	}).Parse(messageTemplate)

	if err != nil {
		return nil, nil, errorutil.Wrap(err)
	}

	message, err := core.TranslateNotification(n, translator)
	if err != nil {
		return nil, nil, errorutil.Wrap(err)
	}

	date := clock.Now().Format(time.RFC1123Z)

	recipients, err := func() ([]string, error) {
		a, err := mail.ParseAddressList(settings.Recipients)
		if err != nil {
			return []string{}, errorutil.Wrap(err)
		}

		r := []string{}
		for _, v := range a {
			r = append(r, v.Address)
		}

		return r, nil
	}()

	if err != nil {
		return nil, nil, errorutil.Wrap(err)
	}

	headers := map[string]string{
		"To":                        settings.Recipients,
		"From":                      settings.Sender,
		"Date":                      date,
		"Subject":                   message.Title,
		"User-Agent":                fmt.Sprintf("Lightmeter ControlCenter %v (%v)", version.Version, version.Commit),
		"MIME-Version":              "1.0",
		"Content-Type":              "text/html; charset=UTF-8",
		"Content-Language":          "en-US", // TODO: use language of the translator
		"Content-Transfer-Encoding": "7bit",
	}

	payload, err := func() (string, error) {
		var b strings.Builder

		for k, v := range headers {
			b.WriteString(k)
			b.WriteString(": ")
			b.WriteString(v)
			b.WriteString("\r\n")
		}

		b.WriteString("\r\n")

		err := template.Execute(&b, buildTemplateValues(n.ID, message, globalSettings))

		if err != nil {
			return "", errorutil.Wrap(err)
		}

		b.WriteString("\r\n")

		return b.String(), nil
	}()

	if err != nil {
		return nil, nil, errorutil.Wrap(err)
	}

	reader := strings.NewReader(payload)

	return reader, recipients, nil
}

func validateEmail(s string) error {
	if strings.Contains(s, "\r\n") {
		return ErrInvalidEmail
	}

	return nil
}

// implement Notifier
// TODO: split this function into smaller chunks!!!
func (m *Notifier) Notify(n core.Notification, translator translator.Translator) error {
	reject, err := m.policy.Reject(n)
	if err != nil {
		return errorutil.Wrap(err)
	}

	if reject {
		return nil
	}

	settings, globalSettings, err := m.settingsFetcher()
	if err != nil {
		return errorutil.Wrap(err)
	}

	if settings == nil || globalSettings == nil {
		panic("Settings cannot be nil!")
	}

	onClient := func(c *smtp.Client) error {
		if err := validateEmail(settings.Sender); err != nil {
			return errorutil.Wrap(err)
		}

		if err := validateEmail(settings.Recipients); err != nil {
			return errorutil.Wrap(err)
		}

		bodyReader, recipients, err := buildMessageProperties(translator, n, m.clock, settings, globalSettings)
		if err != nil {
			return errorutil.Wrap(err)
		}

		if err := c.Mail(settings.Sender, nil); err != nil {
			return errorutil.Wrap(err)
		}

		for _, recipient := range recipients {
			if err := c.Rcpt(recipient); err != nil {
				return errorutil.Wrap(err)
			}
		}

		w, err := c.Data()
		if err != nil {
			return errorutil.Wrap(err)
		}

		_, err = io.Copy(w, bodyReader)
		if err != nil {
			return errorutil.Wrap(err)
		}

		if err := w.Close(); err != nil {
			return errorutil.Wrap(err)
		}

		return nil
	}

	if err := sendOnClient(*settings, onClient); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func (*Notifier) ValidateSettings(s core.Settings) error {
	settings, ok := s.(Settings)

	if !ok {
		return core.ErrInvalidSettings
	}

	if err := ValidateSettings(settings); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func SetSettings(ctx context.Context, writer *metadata.AsyncWriter, settings Settings) error {
	if err := writer.StoreJsonSync(ctx, SettingKey, settings); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func GetSettings(ctx context.Context, reader metadata.Reader) (*Settings, error) {
	settings := &Settings{}

	err := reader.RetrieveJson(ctx, SettingKey, settings)
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	return settings, nil
}
