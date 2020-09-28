package httpmiddleware

import (
	"context"
	"errors"
	"gitlab.com/lightmeter/controlcenter/data"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"net/http"
	"net/url"
	"time"
)

type Interval string

func GetIntervalFromContext(r *http.Request) data.TimeInterval {
	ti, err := getIntervalFromContext(r.Context())
	if err != nil {
		panic(err)
	}

	return ti
}

func getIntervalFromContext(ctx context.Context) (data.TimeInterval, error) {
	interval, ok := ctx.Value(Interval("interval")).(data.TimeInterval)
	if !ok {
		return data.TimeInterval{}, errors.New("interval value is bad or missing")
	}

	return interval, nil
}

func RequestWithInterval(timezone *time.Location) func(h http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.ParseForm() != nil {
				http.Error(w, "Wrong input", http.StatusUnprocessableEntity)
				return
			}

			interval, err := intervalFromForm(timezone, r.Form)

			if err != nil {
				http.Error(w, "Error parsing time interval:\""+err.Error()+"\"", http.StatusUnprocessableEntity)
				return
			}

			ctx := r.Context()
			ctx = context.WithValue(ctx, Interval("interval"), interval)
			r = r.WithContext(ctx)

			h.ServeHTTP(w, r)
		})
	}
}

func intervalFromForm(timezone *time.Location, form url.Values) (data.TimeInterval, error) {
	interval, err := data.ParseTimeInterval(form.Get("from"), form.Get("to"), timezone)

	if err != nil {
		return data.TimeInterval{}, errorutil.Wrap(err)
	}

	return interval, nil
}
