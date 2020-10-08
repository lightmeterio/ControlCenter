package httpmiddleware

import (
	"context"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"net/http"
	"time"
)

const DefaultTimeout = time.Second * 30

func WithDefaultTimeout(middleware ...Middleware) Chain {
	return WithTimeout(DefaultTimeout, middleware...)
}

func WithTimeout(timeout time.Duration, middleware ...Middleware) Chain {
	middleware = append(middleware, []Middleware{RequestWithTimeout(timeout)}...)
	return New(middleware...)
}

func RequestWithTimeout(timeout time.Duration) Middleware {
	return func(h CustomHTTPHandler) CustomHTTPHandler {
		return CustomHTTPHandler(func(w http.ResponseWriter, r *http.Request) error {
			now := time.Now()

			ctx, cancel := context.WithTimeout(r.Context(), timeout)

			defer cancel()

			err := h.ServeHTTP(w, r.WithContext(ctx))

			if deadline, ok := ctx.Deadline(); ok && ctx.Err() != nil {
				elapsedTime := deadline.Sub(now)
				return NewHTTPStatusCodeError(http.StatusRequestTimeout, errorutil.Wrap(err, "HTTP request", r.URL.Redacted(), "with timeout of", elapsedTime))
			}

			return nil
		})
	}
}
