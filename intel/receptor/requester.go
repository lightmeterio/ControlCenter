// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package receptor

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog/log"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"net/http"
	"net/url"
	"time"
)

type HTTPRequester struct {
	URL     string
	Timeout time.Duration
}

func (r *HTTPRequester) Request(ctx context.Context, payload Payload) (event *Event, err error) {
	encodedArgs := url.Values{
		"instance-id": []string{payload.InstanceID},
		"event-id":    []string{payload.LastKnownEventID},
	}.Encode()

	withTimeout, cancel := context.WithTimeout(ctx, r.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(withTimeout, http.MethodGet, fmt.Sprintf("%s?%s", r.URL, encodedArgs), nil)
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	req.Header.Set("Content-Type", "application/json")

	//nolint:bodyclose
	response, err := (&http.Client{}).Do(req)
	if err != nil {
		log.Error().Err(err).Msg("Could not obtain event")
		return nil, nil
	}

	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	defer errorutil.UpdateErrorFromCloser(response.Body, &err)

	var ev Event
	if err := json.NewDecoder(response.Body).Decode(&ev); err != nil {
		log.Error().Err(err).Msg("Could not parse event payload")
		return nil, nil
	}

	return &ev, nil
}
