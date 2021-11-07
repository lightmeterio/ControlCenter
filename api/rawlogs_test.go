// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package api

import (
	"compress/gzip"
	"context"
	"github.com/gorilla/sessions"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/httpauth"
	"gitlab.com/lightmeter/controlcenter/httpauth/auth"
	"gitlab.com/lightmeter/controlcenter/metadata"
	"gitlab.com/lightmeter/controlcenter/rawlogsdb"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

type fakeRawLogsFetcher struct {
}

func (f *fakeRawLogsFetcher) FetchLogsInInterval(ctx context.Context, interval timeutil.TimeInterval, pageSize int, cursor int64) (rawlogsdb.Content, error) {
	return rawlogsdb.Content{}, nil
}

func (f *fakeRawLogsFetcher) FetchLogsInIntervalToWriter(ctx context.Context, interval timeutil.TimeInterval, w io.Writer) error {
	w.Write([]byte("log line 1\nlog line 2\n"))
	return nil
}

func (f *fakeRawLogsFetcher) CountLogLinesInInterval(context.Context, timeutil.TimeInterval) (int64, error) {
	return 0, nil
}

func TestFetchingRawLogs(t *testing.T) {
	Convey("Fetch Raw Logs", t, func() {
		registrar := &auth.FakeRegistrar{
			Email:      "alice@example.com",
			Password:   "super_secret",
			SessionKey: []byte("AAAAAAAAAAAAAAAA"),
		}

		authenticator := &auth.Authenticator{
			Registrar: registrar,
			Store:     sessions.NewCookieStore([]byte("secret-key")),
		}

		settingdDB, removeDB := testutil.TempDBConnectionMigrated(t, "master")
		defer removeDB()

		handler, err := metadata.NewHandler(settingdDB)
		So(err, ShouldBeNil)

		fetcher := &fakeRawLogsFetcher{}

		mux := http.NewServeMux()
		HttpRawLogs(authenticator, mux, time.UTC, fetcher)

		httpauth.HttpAuthenticator(mux, authenticator, handler.Reader, true)

		s := httptest.NewServer(mux)

		httpClient := buildCookieClient()

		Convey("Unauthorized access", func() {
			r, err := httpClient.Get(s.URL + "/api/v0/fetchRawLogsInTimeInterval?from=2000-01-01&to=4000-01-01&format=plain")
			So(err, ShouldBeNil)
			So(r.StatusCode, ShouldEqual, http.StatusUnauthorized)
		})

		Convey("Authorized access", func() {
			r, err := httpClient.PostForm(s.URL+"/login", url.Values{"email": {"alice@example.com"}, "password": {"super_secret"}})
			So(r.StatusCode, ShouldEqual, http.StatusOK)
			So(err, ShouldBeNil)

			Convey("Fetch plain logs", func() {
				r, err := httpClient.Get(s.URL + "/api/v0/fetchRawLogsInTimeInterval?from=2000-01-01&to=4000-01-01&format=plain")
				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusOK)
				So(r.Header.Get("Content-Type"), ShouldEqual, "text/plain")

				content, err := io.ReadAll(r.Body)
				So(err, ShouldBeNil)

				So(content, ShouldResemble, []byte("log line 1\nlog line 2\n"))
			})

			Convey("Fetch gzipped logs", func() {
				r, err := httpClient.Get(s.URL + "/api/v0/fetchRawLogsInTimeInterval?from=2000-01-01&to=4000-01-01")
				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusOK)
				So(r.Header.Get("Content-Type"), ShouldEqual, "application/gzip")

				decompressor, err := gzip.NewReader(r.Body)
				So(err, ShouldBeNil)

				content, err := io.ReadAll(decompressor)
				So(err, ShouldBeNil)

				So(content, ShouldResemble, []byte("log line 1\nlog line 2\n"))
			})
		})
	})
}
