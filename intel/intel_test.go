// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package intel

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	_ "gitlab.com/lightmeter/controlcenter/insights/migrations"
	"gitlab.com/lightmeter/controlcenter/intel/collector"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/meta"
	"gitlab.com/lightmeter/controlcenter/settings/globalsettings"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
)

func init() {
	lmsqlite3.Initialize(lmsqlite3.Options{})
}

type fakeReportServerHandler struct {
	response map[string]interface{}
}

func (h *fakeReportServerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)

	v := map[string]interface{}{}

	errorutil.MustSucceed(decoder.Decode(&v))

	h.response = v
}

func fakeVersion() Version {
	return Version{Version: "1.0", TagOrBranch: "some_branch", Commit: "123456"}
}

func TestReports(t *testing.T) {
	Convey("Test Reports", t, func() {
		handler := &fakeReportServerHandler{}

		s := httptest.NewServer(handler)

		conn, clear := testutil.TempDBConnection(t)

		defer clear()

		m, err := meta.NewHandler(conn, "master")
		So(err, ShouldBeNil)

		Convey("Server error should not cause the dispatching to fail", func() {
			err = (&Dispatcher{
				versionBuilder:       fakeVersion,
				ReportDestinationURL: "http://completely_wrong_url",
				SettingsReader:       m.Reader,
			}).Dispatch(collector.Report{})

			So(err, ShouldBeNil)

			So(len(handler.response), ShouldEqual, 0)
		})

		Convey("Send settings if available", func() {
			err := m.Writer.StoreJson(context.Background(), globalsettings.SettingKey, globalsettings.Settings{
				LocalIP:     net.ParseIP(`127.0.0.2`),
				APPLanguage: "en",
				PublicURL:   "https://example.com",
			})

			So(err, ShouldBeNil)

			err = (&Dispatcher{
				versionBuilder:       fakeVersion,
				ReportDestinationURL: s.URL,
				SettingsReader:       m.Reader,
			}).Dispatch(collector.Report{
				Interval: timeutil.TimeInterval{From: timeutil.MustParseTime(`2000-01-01 00:00:00 +0000`), To: timeutil.MustParseTime(`2000-01-01 10:00:00 +0000`)},
				Content: []collector.ReportEntry{
					{Time: timeutil.MustParseTime(`2000-01-01 01:00:00 +0000`), ID: "some_id", Payload: "some_payload"},
				},
			})

			So(err, ShouldBeNil)

			So(handler.response, ShouldResemble, map[string]interface{}{
				"metadata": map[string]interface{}{
					"postfix_public_ip": "127.0.0.2",
					"public_url":        "https://example.com",
				},
				"app_version": map[string]interface{}{"version": "1.0", "tag_or_branch": "some_branch", "commit": "123456"},
				"payload": map[string]interface{}{
					"interval": map[string]interface{}{
						"from": "2000-01-01T00:00:00Z",
						"to":   "2000-01-01T10:00:00Z",
					},
					"content": []interface{}{
						map[string]interface{}{
							"time":    "2000-01-01T01:00:00Z",
							"id":      "some_id",
							"payload": "some_payload",
						},
					},
				},
			})
		})

		Convey("Do not send settings if not available", func() {
			err = (&Dispatcher{
				versionBuilder:       fakeVersion,
				ReportDestinationURL: s.URL,
				SettingsReader:       m.Reader,
			}).Dispatch(collector.Report{
				Interval: timeutil.TimeInterval{From: timeutil.MustParseTime(`2000-01-01 00:00:00 +0000`), To: timeutil.MustParseTime(`2000-01-01 10:00:00 +0000`)},
				Content: []collector.ReportEntry{
					{Time: timeutil.MustParseTime(`2000-01-01 01:00:00 +0000`), ID: "some_id", Payload: "some_payload"},
				},
			})

			So(err, ShouldBeNil)

			So(handler.response, ShouldResemble, map[string]interface{}{
				"metadata":    map[string]interface{}{},
				"app_version": map[string]interface{}{"version": "1.0", "tag_or_branch": "some_branch", "commit": "123456"},
				"payload": map[string]interface{}{
					"interval": map[string]interface{}{
						"from": "2000-01-01T00:00:00Z",
						"to":   "2000-01-01T10:00:00Z",
					},
					"content": []interface{}{
						map[string]interface{}{
							"time":    "2000-01-01T01:00:00Z",
							"id":      "some_id",
							"payload": "some_payload",
						},
					},
				},
			})
		})
	})
}
