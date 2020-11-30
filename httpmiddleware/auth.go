package httpmiddleware

import (
	"context"
	"fmt"
	"github.com/gorilla/sessions"
	v2 "gitlab.com/lightmeter/controlcenter/httpauth/v2"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"net/http"
	"time"
)

type SessionName string

const SessionKey SessionName = v2.SessionName

func GetSession(ctx context.Context) *sessions.Session {
	return ctx.Value(SessionKey).(*sessions.Session)
}

func RequestWithSession(auth *v2.Authenticator) Middleware {
	return func(h CustomHTTPHandler) CustomHTTPHandler {
		return CustomHTTPHandler(func(w http.ResponseWriter, r *http.Request) error {

			session, err := auth.Store.Get(r, v2.SessionName)
			if err != nil {
				cookie := &http.Cookie{
					Name:     v2.SessionName,
					Value:    "",
					Path:     "/",
					Expires:  time.Unix(0, 0),
					HttpOnly: true,
				}
				http.SetCookie(w, cookie)

				return fmt.Errorf("Error getting session. Force session expiration: %w", errorutil.Wrap(err))
			}

			sessionData, ok := session.Values["auth"].(*v2.SessionData)
			if !(ok && sessionData.IsAuthenticated()) {
				w.WriteHeader(http.StatusUnauthorized)
				return nil
			}

			ctx := context.WithValue(r.Context(), SessionKey, session)

			return h.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
