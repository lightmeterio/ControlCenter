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
	"gitlab.com/lightmeter/controlcenter/auth"
	_ "gitlab.com/lightmeter/controlcenter/insights/migrations"
	"gitlab.com/lightmeter/controlcenter/intel/collector"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/meta"
	"gitlab.com/lightmeter/controlcenter/postfixversion"
	"gitlab.com/lightmeter/controlcenter/settings/globalsettings"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/postfixutil"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"strings"
	"time"
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

		dir, closeDatabases := testutil.TempDatabases(t)
		defer closeDatabases()

		auth, err := auth.NewAuth(dir, auth.Options{})
		So(err, ShouldBeNil)

		runner := meta.NewRunner(dbconn.Db("master"))
		done, cancel := runner.Run()

		defer func() {
			cancel()
			So(done(), ShouldBeNil)
		}()

		email := "user@lightmeter.io"

		_, err = auth.Register(context.Background(), email, "username", "that_password_5689")
		So(err, ShouldBeNil)

		Convey("Server error should not cause the dispatching to fail", func() {
			err = (&Dispatcher{
				VersionBuilder:       fakeVersion,
				ReportDestinationURL: "http://completely_wrong_url",
				Auth:                 auth,
			}).Dispatch(collector.Report{})

			So(err, ShouldBeNil)

			So(len(handler.response), ShouldEqual, 0)
		})

		Convey("Send settings if available", func() {
			err := meta.StoreJson(context.Background(), dbconn.Db("master"), globalsettings.SettingKey, globalsettings.Settings{
				LocalIP:     net.ParseIP(`127.0.0.2`),
				APPLanguage: "en",
				PublicURL:   "https://example.com",
			})

			So(err, ShouldBeNil)

			err = (&Dispatcher{
				InstanceID:           "my-best-uuid",
				VersionBuilder:       fakeVersion,
				ReportDestinationURL: s.URL,
				Auth:                 auth,
			}).Dispatch(collector.Report{
				Interval: timeutil.TimeInterval{From: timeutil.MustParseTime(`2000-01-01 00:00:00 +0000`), To: timeutil.MustParseTime(`2000-01-01 10:00:00 +0000`)},
				Content: []collector.ReportEntry{
					{Time: timeutil.MustParseTime(`2000-01-01 01:00:00 +0000`), ID: "some_id", Payload: "some_payload"},
				},
			})

			So(err, ShouldBeNil)

			So(handler.response, ShouldResemble, map[string]interface{}{
				"metadata": map[string]interface{}{
					"instance_id":       "my-best-uuid",
					"postfix_public_ip": "127.0.0.2",
					"public_url":        "https://example.com",
					"user_email":        email,
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
				InstanceID:           "my-best-uuid",
				VersionBuilder:       fakeVersion,
				ReportDestinationURL: s.URL,
				Auth:                 auth,
			}).Dispatch(collector.Report{
				Interval: timeutil.TimeInterval{From: timeutil.MustParseTime(`2000-01-01 00:00:00 +0000`), To: timeutil.MustParseTime(`2000-01-01 10:00:00 +0000`)},
				Content: []collector.ReportEntry{
					{Time: timeutil.MustParseTime(`2000-01-01 01:00:00 +0000`), ID: "some_id", Payload: "some_payload"},
				},
			})

			So(err, ShouldBeNil)

			So(handler.response, ShouldResemble, map[string]interface{}{
				"metadata":    map[string]interface{}{"user_email": email, "instance_id": "my-best-uuid"},
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

		Convey("Send postfix version", func() {
			p := postfixversion.NewPublisher(runner.Writer())
			postfixutil.ReadFromTestReader(strings.NewReader("Mar 29 12:55:50 test1 postfix/master[15019]: daemon started -- version 3.4.14, configuration /etc/postfix"), p, 2000)
			time.Sleep(100 * time.Millisecond)

			err = (&Dispatcher{
				InstanceID:           "my-best-uuid",
				VersionBuilder:       fakeVersion,
				ReportDestinationURL: s.URL,
				Auth:                 auth,
			}).Dispatch(collector.Report{
				Interval: timeutil.TimeInterval{From: timeutil.MustParseTime(`2000-01-01 00:00:00 +0000`), To: timeutil.MustParseTime(`2000-01-01 10:00:00 +0000`)},
				Content: []collector.ReportEntry{
					{Time: timeutil.MustParseTime(`2000-01-01 01:00:00 +0000`), ID: "some_id", Payload: "some_payload"},
				},
			})

			So(err, ShouldBeNil)

			So(handler.response, ShouldResemble, map[string]interface{}{
				"metadata":    map[string]interface{}{"user_email": email, "instance_id": "my-best-uuid", "postfix_version": "3.4.14"},
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
