// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package httpmiddleware

import (
	"context"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	uuid "github.com/satori/go.uuid"
	"gitlab.com/lightmeter/controlcenter/httpauth/auth"
	"gitlab.com/lightmeter/controlcenter/pkg/ctxlogger"
	"gitlab.com/lightmeter/controlcenter/pkg/httperror"
	"gitlab.com/lightmeter/controlcenter/util/httputil"
	"net/http"
)

type CustomHTTPHandlerInterface interface {
	ServeHTTP(w http.ResponseWriter, r *http.Request) error
}

type CustomHTTPHandler func(w http.ResponseWriter, r *http.Request) error

func (f CustomHTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) error {
	return f(w, r)
}

type Middleware func(CustomHTTPHandler) CustomHTTPHandler

type Chain struct {
	middleware []Middleware
}

func WithDefaultStackWithoutAuth(middleware ...Middleware) Chain {
	middleware = append([]Middleware{RequestWithTimeout(DefaultTimeout)}, middleware...)
	return New(middleware...)
}

func WithDefaultStack(auth *auth.Authenticator, middleware ...Middleware) Chain {
	middleware = append([]Middleware{RequestWithTimeout(DefaultTimeout), RequestWithSession(auth)}, middleware...)
	return New(middleware...)
}

func New(middleware ...Middleware) Chain {
	return Chain{middleware: middleware}
}

func (c Chain) WithEndpoint(endpoint CustomHTTPHandlerInterface) http.Handler {
	if endpoint == nil {
		panic("endpoint is nil")
	}

	for i := range c.middleware {
		if c.middleware == nil {
			panic("middleware is nil")
		}

		endpoint = c.middleware[len(c.middleware)-1-i](endpoint.ServeHTTP)
	}

	return wrapWithErrorHandler(endpoint)
}

func (c Chain) WithError(endpoint CustomHTTPHandlerInterface) http.Handler {
	if endpoint == nil {
		panic("endpoint is nil")
	}

	return wrapWithErrorHandler(endpoint)
}

type RequestID string

const RequestIDKey RequestID = "RequestIDKey"
const LoggerKey string = "LoggerKey"

func wrapWithErrorHandler(endpoint CustomHTTPHandlerInterface) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		requestId := uuid.NewV4().String()

		ctx := context.WithValue(r.Context(), RequestIDKey, requestId)
		r = r.WithContext(ctx)

		logger := LoggerWithHTTPContext(requestId)

		//nolint:golint,staticcheck
		ctx = context.WithValue(r.Context(), LoggerKey, &logger)
		r = r.WithContext(ctx)

		err := endpoint.ServeHTTP(w, r)

		//nolint:errorlint
		switch errType := err.(type) {
		case httperror.XHTTPError:
			if errType.StatusCode() >= 500 {
				ctxlogger.LogErrorf(r.Context(), errType.ErrObject(), "Internal server error")
				w.WriteHeader(errType.StatusCode())
				return
			}

			if errType.JSON() {
				response := struct {
					Error string
				}{
					Error: errType.Error(),
				}

				if err := httputil.WriteJson(w, response, errType.StatusCode()); err != nil {
					ctxlogger.LogErrorf(r.Context(), err, "Internal server error")
					w.WriteHeader(http.StatusInternalServerError)
				}
				return
			}

			http.Error(w, errType.Error(), errType.StatusCode())

		case error:
			ctxlogger.LogErrorf(r.Context(), err, "Internal server error")
			w.WriteHeader(http.StatusInternalServerError)
		}
	})
}

// Logger returns a logger with http context
func LoggerWithHTTPContext(requestId string) zerolog.Logger {
	newLogger := log.Logger

	if requestId != "" {
		newLogger.UpdateContext(func(c zerolog.Context) zerolog.Context {
			return c.Str("requestID", requestId)
		})
	}

	return newLogger
}
