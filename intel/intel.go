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
	"gitlab.com/lightmeter/controlcenter/deliverydb"
	"gitlab.com/lightmeter/controlcenter/intel/collector"
	"gitlab.com/lightmeter/controlcenter/intel/mailactivity"
	"gitlab.com/lightmeter/controlcenter/meta"
	"gitlab.com/lightmeter/controlcenter/settings/globalsettings"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"net/http"
	"time"
)

type Metadata struct {
	LocalIP   *string `json:"postfix_public_ip,omitempty"`
	PublicURL *string `json:"public_url,omitempty"`
}

type ReportWithMetadata struct {
	Metadata Metadata         `json:"metadata"`
	Payload  collector.Report `json:"payload"`
}

type Dispatcher struct {
	ReportDestinationURL string
	SettingsReader       *meta.Reader
}

func (d *Dispatcher) Dispatch(r collector.Report) error {
	log.Info().Msgf("Sending a new Network intelligence report in the interval %v and with %v rows", r.Interval, len(r.Content))

	metadata, err := func() (Metadata, error) {
		settings, err := globalsettings.GetSettings(context.Background(), d.SettingsReader)
		if err != nil && errors.Is(err, meta.ErrNoSuchKey) {
			return Metadata{}, nil
		}

		if err != nil {
			return Metadata{}, errorutil.Wrap(err)
		}

		addr := func(s string) *string {
			return &s
		}

		ip := func() *string {
			if settings.LocalIP != nil {
				return addr(settings.LocalIP.String())
			}

			return nil
		}()

		return Metadata{LocalIP: ip, PublicURL: addr(settings.PublicURL)}, nil
	}()

	if err != nil {
		return errorutil.Wrap(err)
	}

	reportWithMetadata := ReportWithMetadata{
		Metadata: metadata,
		Payload:  r,
	}

	json, err := json.Marshal(reportWithMetadata)
	if err != nil {
		return errorutil.Wrap(err)
	}

	ctx, cancelCtx := context.WithTimeout(context.Background(), 2500*time.Millisecond)

	defer cancelCtx()

	req, err := http.NewRequestWithContext(ctx, "POST", d.ReportDestinationURL, bytes.NewBuffer(json))
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

	if err := response.Body.Close(); err != nil {
		// Not a fatal error; just ignore it
		log.Err(err).Msgf("Error closing response body")
		return nil
	}

	return nil
}

type Options struct {
	// How often should the c
	CycleInterval time.Duration

	// How often should the reports be dispatched/sent?
	ReportInterval time.Duration

	ReportDestinationURL string
}

func New(workspaceDir string, db *deliverydb.DB, settingsReader *meta.Reader, options Options) (*collector.Collector, error) {
	reporters := collector.Reporters{
		mailactivity.NewReporter(db.ConnPool()),
	}

	collectorOptions := collector.Options{
		CycleInterval:  options.CycleInterval,
		ReportInterval: options.ReportInterval,
	}

	dispatcher := &Dispatcher{
		SettingsReader:       settingsReader,
		ReportDestinationURL: options.ReportDestinationURL,
	}

	c, err := collector.New(workspaceDir, collectorOptions, reporters, dispatcher)
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	return c, nil
}
