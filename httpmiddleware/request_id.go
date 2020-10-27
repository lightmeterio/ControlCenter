package httpmiddleware

import (
	"context"
	uuid "github.com/satori/go.uuid"
	"net/http"
)

type RequestID string

const RequestIDKey RequestID = "RequestIDKey"

func RequestWithID() Middleware {
	return func(h CustomHTTPHandler) CustomHTTPHandler {
		return CustomHTTPHandler(func(w http.ResponseWriter, r *http.Request) error {

			ctx := context.WithValue(r.Context(), RequestIDKey, uuid.NewV4().String())

			return h.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetRequestID(ctx context.Context) string {
	return ctx.Value(RequestIDKey).(string)
}
