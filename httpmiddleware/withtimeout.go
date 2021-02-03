// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package httpmiddleware

import (
	"context"
	"errors"
	"gitlab.com/lightmeter/controlcenter/pkg/httperror"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"net/http"
	"regexp"
	"strconv"
	"time"
)

const (
	DefaultTimeout   = time.Second * 30
	MaxCustomTimeout = time.Minute * 1
)

var (
	// NOTE: this is not an exhaustive parser for Keep-Alive. It's just good enough for our needs
	// More information about Keep-Alive at https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Keep-Alive
	keepAliveRegexp = regexp.MustCompile(`^timeout=(\d+)(, max=\d+)?$`)

	ErrInvalidKeepAliveHeader = errors.New("Could not parse Keep-Alive Header")
)

func timeoutForRequest(r *http.Request, defaultTimeout, maxTimeout time.Duration) (time.Duration, error) {
	keepAlive := r.Header.Get("Keep-Alive")

	if len(keepAlive) == 0 {
		return defaultTimeout, nil
	}

	matches := keepAliveRegexp.FindSubmatch([]byte(keepAlive))

	if len(matches) == 0 {
		return 0, ErrInvalidKeepAliveHeader
	}

	seconds, err := strconv.Atoi(string(matches[1]))

	if err != nil {
		return 0, errorutil.Wrap(err)
	}

	timeout := time.Second * time.Duration(seconds)

	if timeout > maxTimeout {
		return maxTimeout, nil
	}

	return timeout, nil
}

func RequestWithTimeout(defaultTimeout time.Duration) Middleware {
	return func(h CustomHTTPHandler) CustomHTTPHandler {
		return CustomHTTPHandler(func(w http.ResponseWriter, r *http.Request) error {
			now := time.Now()

			timeout, err := timeoutForRequest(r, defaultTimeout, MaxCustomTimeout)

			if err != nil {
				return httperror.NewHTTPStatusCodeError(http.StatusBadRequest, errorutil.Wrap(err, "Error reading Keep-Alive header"))
			}

			ctx, cancel := context.WithTimeout(r.Context(), timeout)

			defer cancel()

			err = h.ServeHTTP(w, r.WithContext(ctx))

			if deadline, ok := ctx.Deadline(); ok && ctx.Err() != nil {
				elapsedTime := deadline.Sub(now)
				return httperror.NewHTTPStatusCodeError(http.StatusRequestTimeout, errorutil.Wrap(err, "HTTP request", r.URL.Redacted(), "with timeout of", elapsedTime))
			}

			return err
		})
	}
}
