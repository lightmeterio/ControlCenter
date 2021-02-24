// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package httpmiddleware

import (
	"context"
	"errors"
	"gitlab.com/lightmeter/controlcenter/pkg/httperror"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"net/http"
	"net/url"
	"time"
)

type Interval string

func GetIntervalFromContext(r *http.Request) timeutil.TimeInterval {
	ti, err := getIntervalFromContext(r.Context())
	if err != nil {
		panic(err)
	}

	return ti
}

func getIntervalFromContext(ctx context.Context) (timeutil.TimeInterval, error) {
	interval, ok := ctx.Value(Interval("interval")).(timeutil.TimeInterval)
	if !ok {
		return timeutil.TimeInterval{}, errors.New("interval value is bad or missing")
	}

	return interval, nil
}

func RequestWithInterval(timezone *time.Location) Middleware {
	return func(h CustomHTTPHandler) CustomHTTPHandler {
		return CustomHTTPHandler(func(w http.ResponseWriter, r *http.Request) error {
			if r.ParseForm() != nil {
				return httperror.NewHTTPStatusCodeError(http.StatusUnprocessableEntity, errors.New("Wrong Input"))
			}

			interval, err := intervalFromForm(timezone, r.Form)

			if err != nil {
				return httperror.NewHTTPStatusCodeError(http.StatusUnprocessableEntity, errors.New("Error parsing time interval:\""+err.Error()+"\""))
			}
			ctx := r.Context()
			ctx = context.WithValue(ctx, Interval("interval"), interval)
			r = r.WithContext(ctx)

			return h.ServeHTTP(w, r)
		})
	}
}

func intervalFromForm(timezone *time.Location, form url.Values) (timeutil.TimeInterval, error) {
	interval, err := timeutil.ParseTimeInterval(form.Get("from"), form.Get("to"), timezone)

	if err != nil {
		return timeutil.TimeInterval{}, errorutil.Wrap(err)
	}

	return interval, nil
}
