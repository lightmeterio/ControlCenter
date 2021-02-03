// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package newsletter

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"github.com/rs/zerolog/log"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

var ErrSubscribingToNewsletter = errors.New(`Failed to Subscribe to newsletter`)

type Subscriber interface {
	Subscribe(context context.Context, email string) error
}

type HTTPSubscriber struct {
	URL        string
	HTTPClient *http.Client
}

func encodeBody(reader io.Reader) (string, error) {
	body, err := ioutil.ReadAll(reader)

	if err != nil {
		return "", errorutil.Wrap(err)
	}

	var base64Writer bytes.Buffer

	encoder := base64.NewEncoder(base64.StdEncoding, &base64Writer)

	defer func() {
		if err := encoder.Close(); err != nil {
			errorutil.LogErrorf(errorutil.Wrap(err), "could not close base64 encoder")
		}
	}()

	if _, err := encoder.Write(body); err != nil {
		return "", errorutil.Wrap(err)
	}

	return base64Writer.String(), nil
}

func (s *HTTPSubscriber) Subscribe(context context.Context, email string) error {
	data := url.Values{}

	data.Set("email", email)
	data.Set("htmlemail", "1")
	data.Set("list[11]", "signup")
	data.Set("subscribe", "subscribe")

	content := strings.NewReader(data.Encode())

	req, err := http.NewRequestWithContext(context, http.MethodPost, s.URL+"/lists/?p=asubscribe&id=2", content)

	if err != nil {
		return errorutil.Wrap(err)
	}

	req.Header.Set("Content-Type", `application/x-www-form-urlencoded; charset=UTF-8`)

	res, err := s.HTTPClient.Do(req)

	if err != nil {
		return errorutil.Wrap(err)
	}

	defer func() { errorutil.MustSucceed(res.Body.Close()) }()

	body, err := encodeBody(res.Body)

	if err != nil {
		return errorutil.Wrap(err)
	}

	log.Info().Msgf("Subscribe response: %v", body)

	if res.StatusCode != http.StatusOK {
		log.Error().Msgf("subscribing email to newsletter failed!")
		return errorutil.Wrap(ErrSubscribingToNewsletter)
	}

	log.Info().Msg("Successfully subscribed email to newsletter!")

	return nil
}
