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
)

type HTTPRequester struct {
	URL string
}

func (r *HTTPRequester) Request(ctx context.Context, payload Payload) (*Event, error) {
	encodedArgs := url.Values{
		"instance-id": []string{payload.InstanceID},
		"event-id":    []string{payload.LastKnownEventID},
	}.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s?%s", r.URL, encodedArgs), nil)
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	req.Header.Set("Content-Type", "application/json")

	response, err := (&http.Client{}).Do(req)
	if err != nil {
		log.Error().Err(err).Msg("Could not obtain event")
		return nil, nil
	}

	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	defer func() {
		// TODO: handle it once !852 is implemented
		errorutil.MustSucceed(response.Body.Close())
	}()

	var event Event
	if err := json.NewDecoder(response.Body).Decode(&event); err != nil {
		log.Error().Err(err).Msg("Could not parse event payload")
		return nil, nil
	}

	return &event, nil
}
