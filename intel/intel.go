// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package intel

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/rs/zerolog/log"
	"gitlab.com/lightmeter/controlcenter/auth"
	"gitlab.com/lightmeter/controlcenter/connectionstats"
	"gitlab.com/lightmeter/controlcenter/deliverydb"
	"gitlab.com/lightmeter/controlcenter/insights/core"
	"gitlab.com/lightmeter/controlcenter/intel/collector"
	intelConnectionStats "gitlab.com/lightmeter/controlcenter/intel/connectionstats"
	"gitlab.com/lightmeter/controlcenter/intel/insights"
	"gitlab.com/lightmeter/controlcenter/intel/logslinecount"
	"gitlab.com/lightmeter/controlcenter/intel/mailactivity"
	"gitlab.com/lightmeter/controlcenter/intel/topdomains"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/metadata"
	"gitlab.com/lightmeter/controlcenter/postfixversion"
	"gitlab.com/lightmeter/controlcenter/settings/globalsettings"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/version"
	"io"
	"net/http"
	"time"
)

type Metadata struct {
	InstanceID     string  `json:"instance_id"`
	LocalIP        *string `json:"postfix_public_ip,omitempty"`
	PublicURL      *string `json:"public_url,omitempty"`
	UserEmail      *string `json:"user_email,omitempty"`
	PostfixVersion *string `json:"postfix_version,omitempty"`
	MailKind       *string `json:"mail_kind,omitempty"`
}

type Version struct {
	Version     string `json:"version"`
	TagOrBranch string `json:"tag_or_branch"`
	Commit      string `json:"commit"`
}

type ReportWithMetadata struct {
	Metadata Metadata         `json:"metadata"`
	Version  Version          `json:"app_version"`
	Payload  collector.Report `json:"payload"`
}

type Dispatcher struct {
	InstanceID           string
	VersionBuilder       func() Version
	ReportDestinationURL string
	SettingsReader       *metadata.Reader
	Auth                 auth.Registrar
}

func (d *Dispatcher) Dispatch(r collector.Report) error {
	log.Info().Msgf("Sending a new Network intelligence report in the interval %v and with %v rows", r.Interval, len(r.Content))

	metadata, err := func() (Metadata, error) {
		// InstanceID is always available
		metadata := Metadata{InstanceID: d.InstanceID}

		// there can be a postfix version even with no user registered
		metadata.PostfixVersion = d.getPostfixVersion()

		userData, err := d.Auth.GetFirstUser(context.Background())
		if err != nil && !errors.Is(err, auth.ErrNoUser) { // if no user is registered, simply don't send any email
			return metadata, errorutil.Wrap(err)
		}

		if err == nil {
			metadata.UserEmail = &userData.Email
		}

		metadata.PublicURL, metadata.LocalIP, metadata.MailKind = d.getGlobalSettings()

		return metadata, nil
	}()

	if err != nil {
		return errorutil.Wrap(err)
	}

	reportWithMetadata := ReportWithMetadata{
		Version:  d.VersionBuilder(),
		Metadata: metadata,
		Payload:  r,
	}

	json, err := json.Marshal(reportWithMetadata)
	if err != nil {
		return errorutil.Wrap(err)
	}

	ctx, cancelCtx := context.WithTimeout(context.Background(), 2500*time.Millisecond)

	defer cancelCtx()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, d.ReportDestinationURL, bytes.NewBuffer(json))
	if err != nil {
		return errorutil.Wrap(err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := http.Client{}

	response, err := client.Do(req)
	if err != nil {
		log.Err(err).Msgf("Could not send report")

		// NOTE: a network error is not a hard error
		// TODO: maybe retry until it succeeds?
		return nil
	}

	defer func() {
		if err := response.Body.Close(); err != nil {
			// Not a fatal error; just ignore it
			log.Err(err).Msgf("Error closing response body")
		}
	}()

	if response.StatusCode != http.StatusOK {
		body, err := io.ReadAll(response.Body)
		if err != nil {
			log.Err(err).Msgf("Could not send report")
			return nil
		}

		log.Error().Msgf(`Request failed with error code %v and error: %v`, response.Status, string(body))
	}

	return nil
}

func (d *Dispatcher) getGlobalSettings() (*string, *string, *string) {
	settings, err := globalsettings.GetSettings(context.Background(), d.SettingsReader)

	if err != nil && errors.Is(err, metadata.ErrNoSuchKey) {
		log.Warn().Msgf("Unexpected error retrieving global settings")
	}

	if err != nil {
		return nil, nil, nil
	}

	addr := func(s string) *string {
		return &s
	}

	publicURL := func() *string {
		if settings.PublicURL != "" {
			return addr(settings.PublicURL)
		}

		return nil
	}()

	localIP := func() *string {
		if settings.LocalIP != nil {
			return addr(settings.LocalIP.String())
		}

		return nil
	}()

	mailKind := func() *string {
		mailKind, err := d.SettingsReader.Retrieve(context.Background(), "mail_kind")

		if err != nil && !errors.Is(err, metadata.ErrNoSuchKey) {
			log.Warn().Msgf("Unexpected error retrieving mail_kind")
		}

		if err != nil {
			return nil
		}

		s, ok := mailKind.(string)

		if !ok {
			log.Warn().Msgf("mail_kind couldn't be cast to string")
			return nil
		}

		return &s
	}()

	return publicURL, localIP, mailKind
}

func (d *Dispatcher) getPostfixVersion() *string {
	var version string
	err := d.SettingsReader.RetrieveJson(context.Background(), postfixversion.SettingKey, &version)

	if err != nil && !errors.Is(err, metadata.ErrNoSuchKey) {
		log.Warn().Msgf("Unexpected error retrieving postfix version")
	}

	if err != nil {
		return nil
	}

	return &version
}

type Options struct {
	InstanceID string

	// How often should the intel loop should run
	CycleInterval time.Duration

	// How often should the reports be dispatched/sent?
	ReportInterval time.Duration

	ReportDestinationURL string
}

func DefaultVersionBuilder() Version {
	return Version{Version: version.Version, TagOrBranch: version.TagOrBranch, Commit: version.Commit}
}

func New(intelDb *dbconn.PooledPair, deliveryDb *deliverydb.DB, fetcher core.Fetcher,
	settingsReader *metadata.Reader, auth *auth.Auth, connStats *connectionstats.Stats,
	options Options) (*collector.Collector, *logslinecount.Publisher, error) {
	logslinePublisher := logslinecount.NewPublisher()

	reporters := collector.Reporters{
		mailactivity.NewReporter(deliveryDb.ConnPool()),
		insights.NewReporter(fetcher),
		logslinecount.NewReporter(logslinePublisher),
		topdomains.NewReporter(deliveryDb.ConnPool()),
		intelConnectionStats.NewReporter(connStats.ConnPool()),
	}

	collectorOptions := collector.Options{
		CycleInterval:  options.CycleInterval,
		ReportInterval: options.ReportInterval,
	}

	dispatcher := &Dispatcher{
		InstanceID:           options.InstanceID,
		VersionBuilder:       DefaultVersionBuilder,
		SettingsReader:       settingsReader,
		ReportDestinationURL: options.ReportDestinationURL,
		Auth:                 auth,
	}

	c, err := collector.New(intelDb, collectorOptions, reporters, dispatcher)
	if err != nil {
		return nil, nil, errorutil.Wrap(err)
	}

	return c, logslinePublisher, nil
}
