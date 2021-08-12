// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package api

import (
	"context"
	"encoding/json"
	"github.com/gorilla/sessions"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/httpauth/auth"
	"gitlab.com/lightmeter/controlcenter/insights/core"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type fakeProgressFetcher struct {
	value  int
	time   time.Time
	active bool
}

func (f *fakeProgressFetcher) Progress(context.Context) (core.Progress, error) {
	return core.Progress{Value: &f.value, Time: &f.time, Active: f.active}, nil
}

func TestImportProgressEndpoint(t *testing.T) {
	Convey("Test Import Progress", t, func() {
		registrar := &auth.FakeRegistrar{
			SessionKey: []byte("AAAAAAAAAAAAAAAA"),
		}

		authenticator := &auth.Authenticator{
			Registrar: registrar,
			Store:     sessions.NewCookieStore([]byte("secret-key")),
		}

		progressFetcher := &fakeProgressFetcher{}

		mux := http.NewServeMux()
		HttpInsightsProgress(authenticator, mux, progressFetcher)

		s := httptest.NewServer(mux)

		httpClient := buildCookieClient()

		Convey("Obtain progress, no user registered", func() {
			progressFetcher.value = 42
			progressFetcher.time = timeutil.MustParseTime(`2000-01-01 10:00:00 +0000`)
			progressFetcher.active = true

			r, err := httpClient.Get(s.URL + "/api/v0/importProgress")
			So(err, ShouldBeNil)
			So(r.StatusCode, ShouldEqual, http.StatusOK)

			var parsedValue map[string]interface{}
			decoder := json.NewDecoder(r.Body)
			err = decoder.Decode(&parsedValue)
			So(err, ShouldBeNil)

			So(parsedValue, ShouldResemble, map[string]interface{}{
				"value":  float64(42.0),
				"time":   "2000-01-01T10:00:00Z",
				"active": true,
			})
		})

		Convey("Once an user is created, endpoint is not accessible anymore", func() {
			progressFetcher.value = 42
			progressFetcher.time = timeutil.MustParseTime(`2000-01-01 10:00:00 +0000`)
			progressFetcher.active = true

			registrar.Register(context.Background(), "user@example.com", "User Name", "super secret")

			r, err := httpClient.Get(s.URL + "/api/v0/importProgress")
			So(err, ShouldBeNil)
			So(r.StatusCode, ShouldEqual, http.StatusUnauthorized)
		})

	})
}
