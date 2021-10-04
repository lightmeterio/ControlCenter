// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package httpmiddleware

import (
	"context"
	"fmt"
	"github.com/gorilla/sessions"
	"gitlab.com/lightmeter/controlcenter/httpauth/auth"
	httpauth "gitlab.com/lightmeter/controlcenter/httpauth/auth"
	"gitlab.com/lightmeter/controlcenter/pkg/httperror"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"net/http"
	"time"
)

type SessionName string

const SessionKey SessionName = auth.SessionName

func GetSession(ctx context.Context) *sessions.Session {
	return ctx.Value(SessionKey).(*sessions.Session)
}

func RequestWithSession(authenticator *auth.Authenticator) Middleware {
	return func(h CustomHTTPHandler) CustomHTTPHandler {
		return CustomHTTPHandler(func(w http.ResponseWriter, r *http.Request) error {
			session, err := authenticator.Store.Get(r, auth.SessionName)
			if err != nil {
				cookie := &http.Cookie{
					Name:     auth.SessionName,
					Value:    "",
					Path:     "/",
					Expires:  time.Unix(0, 0),
					HttpOnly: true,
				}
				http.SetCookie(w, cookie)

				return fmt.Errorf("Error getting session. Force session expiration: %w", errorutil.Wrap(err))
			}

			sessionData, ok := session.Values["auth"].(*auth.SessionData)
			if !(ok && sessionData.IsAuthenticated()) {
				w.WriteHeader(http.StatusUnauthorized)
				return nil
			}

			ctx := context.WithValue(r.Context(), SessionKey, session)

			return h.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireAuthenticationOnlyAfterSystemHasAnyUser allows something to be seen only if before the user registration ends or the user is authenticated
func RequireAuthenticationOnlyAfterSystemHasAnyUser(auth *httpauth.Authenticator) Middleware {
	return func(h CustomHTTPHandler) CustomHTTPHandler {
		return CustomHTTPHandler(func(w http.ResponseWriter, r *http.Request) error {
			// the progress endpoint can only be accessed in case the user is authenticated
			// or the system registration has not finished yet (aka. no users are registred)
			hasAnyUser, err := auth.Registrar.HasAnyUser(r.Context())
			if err != nil {
				return httperror.NewHTTPStatusCodeError(http.StatusInternalServerError, errorutil.Wrap(err))
			}

			if !hasAnyUser {
				// still pre-registration. Go ahead with the request
				if err := h.ServeHTTP(w, r); err != nil {
					return httperror.NewHTTPStatusCodeError(http.StatusInternalServerError, errorutil.Wrap(err))
				}

				return nil
			}

			// Here the user must be authenticated
			sessionData, err := httpauth.GetSessionData(auth, r)
			if err != nil {
				return httperror.NewHTTPStatusCodeError(http.StatusUnauthorized, errorutil.Wrap(err))
			}

			if !sessionData.IsAuthenticated() {
				return httperror.NewHTTPStatusCodeError(http.StatusUnauthorized, httpauth.ErrUnauthenticated)
			}

			if err := h.ServeHTTP(w, r); err != nil {
				return httperror.NewHTTPStatusCodeError(http.StatusInternalServerError, errorutil.Wrap(err))
			}

			return nil
		})
	}
}
