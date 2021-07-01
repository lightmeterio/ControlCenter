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

	response, err := http.Post(d.ReportDestinationURL, "application/json", bytes.NewBuffer(json))
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

func New(workspaceDir string, db *deliverydb.DB, settingsReader *meta.Reader) (*collector.Collector, error) {
	reporters := collector.Reporters{
		mailactivity.NewReporter(db.ConnPool()),
	}

	options := collector.Options{
		CycleInterval:  time.Minute * 1,
		ReportInterval: time.Minute * 30,
	}

	dispatcher := &Dispatcher{
		SettingsReader:       settingsReader,
		ReportDestinationURL: "https://intel.lightmeter.io/reports",
	}

	c, err := collector.New(workspaceDir, options, reporters, dispatcher)
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	return c, nil
}
