// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package intel

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/auth"
	_ "gitlab.com/lightmeter/controlcenter/insights/migrations"
	"gitlab.com/lightmeter/controlcenter/intel/collector"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/metadata"
	"gitlab.com/lightmeter/controlcenter/newsletter"
	"gitlab.com/lightmeter/controlcenter/pkg/runner"
	"gitlab.com/lightmeter/controlcenter/postfixversion"
	"gitlab.com/lightmeter/controlcenter/settings"
	"gitlab.com/lightmeter/controlcenter/settings/globalsettings"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/postfixutil"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
)

type fakeSchedFile struct {
	reader            *strings.Reader
	shouldFailToRead  bool
	shouldFailToClose bool
}

func (f *fakeSchedFile) Read(p []byte) (int, error) {
	if f.shouldFailToRead {
		return 0, fmt.Errorf("Fake: Failed to read")
	}

	return f.reader.Read(p)
}

func (f *fakeSchedFile) Close() error {
	if f.shouldFailToClose {
		return fmt.Errorf("Fake: Failed to close")
	}

	return nil
}

func fakeSchedReaderFromContent(content string, shouldFailToRead, shouldFailToClose bool) SchedFileReader {
	return func() (io.ReadCloser, error) {
		return &fakeSchedFile{
			reader:            strings.NewReader(content),
			shouldFailToRead:  shouldFailToRead,
			shouldFailToClose: shouldFailToClose,
		}, nil
	}
}

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

		conn, clear := testutil.TempDBConnectionMigrated(t, "master")
		defer clear()

		m, err := metadata.NewHandler(conn)
		So(err, ShouldBeNil)

		writeRunner := metadata.NewSerialWriteRunner(m)
		done, cancel := runner.Run(writeRunner)

		defer func() {
			cancel()
			So(done(), ShouldBeNil)
		}()

		authConn, closeConn := testutil.TempDBConnectionMigrated(t, "auth")
		defer closeConn()

		auth, err := auth.NewAuth(authConn, auth.Options{})
		So(err, ShouldBeNil)

		email := "user@lightmeter.io"
		username := "Jane Doe"

		_, err = auth.Register(context.Background(), email, username, "that_password_5689")
		So(err, ShouldBeNil)

		// notice that the first word in the first line is "lightmeter", which is
		// the main process in the docker container we ship
		// TODO: if one day we change the main binary name used for the docker image,
		// this trick will break!
		schedFileContentForDocker := `lightmeter (1, #threads: 11)
-------------------------------------------------------------------
se.exec_start                                :     149202066.248614
se.vruntime                                  :            24.180579`

		// notice that the first word in the first line is "sh",
		// but when not using our docker image, it could be systemd,
		// or any other init system
		schedFileContentForNonDocker := `systemd (1, #threads: 1)
-------------------------------------------------------------------
se.exec_start                                :     149202066.248614
se.vruntime                                  :            24.180579`

		Convey("Server error should not cause the dispatching to fail", func() {
			err := (&Dispatcher{
				VersionBuilder:       fakeVersion,
				ReportDestinationURL: "http://completely_wrong_url",
				SettingsReader:       m.Reader,
				Auth:                 auth,
				SchedFileReader:      fakeSchedReaderFromContent(schedFileContentForDocker, false, false),
			}).Dispatch(collector.Report{})

			So(err, ShouldBeNil)

			So(len(handler.response), ShouldEqual, 0)
		})

		Convey("Fails to read /proc/1/sched file", func() {
			err := (&Dispatcher{
				VersionBuilder:       fakeVersion,
				ReportDestinationURL: s.URL,
				SettingsReader:       m.Reader,
				Auth:                 auth,
				SchedFileReader:      fakeSchedReaderFromContent(schedFileContentForDocker, true, false),
			}).Dispatch(collector.Report{})

			So(err, ShouldNotBeNil)

			So(len(handler.response), ShouldEqual, 0)
		})

		Convey("Fails to close /proc/1/sched file", func() {
			err := (&Dispatcher{
				VersionBuilder:       fakeVersion,
				ReportDestinationURL: s.URL,
				SettingsReader:       m.Reader,
				Auth:                 auth,
				SchedFileReader:      fakeSchedReaderFromContent(schedFileContentForDocker, false, true),
			}).Dispatch(collector.Report{})

			So(err, ShouldNotBeNil)

			So(len(handler.response), ShouldEqual, 0)
		})

		Convey("Send settings if available", func() {
			err := m.Writer.StoreJson(context.Background(), globalsettings.SettingKey, globalsettings.Settings{
				LocalIP:     globalsettings.IP{net.ParseIP(`127.0.0.2`)},
				AppLanguage: "en",
				PublicURL:   "https://example.com",
			})

			So(err, ShouldBeNil)

			initSettings := settings.NewInitialSetupSettings(&newsletter.FakeNewsletterSubscriber{})

			So(initSettings.Set(context.Background(), writeRunner.Writer(), settings.InitialOptions{SubscribeToNewsletter: false}), ShouldBeNil)

			err = (&Dispatcher{
				InstanceID:           "my-best-uuid",
				VersionBuilder:       fakeVersion,
				ReportDestinationURL: s.URL,
				SettingsReader:       m.Reader,
				Auth:                 auth,
				SchedFileReader:      fakeSchedReaderFromContent(schedFileContentForNonDocker, false, false),
				IsUsingRsyncedLogs:   true,
			}).Dispatch(collector.Report{
				Interval: timeutil.TimeInterval{From: timeutil.MustParseTime(`2000-01-01 00:00:00 +0000`), To: timeutil.MustParseTime(`2000-01-01 10:00:00 +0000`)},
				Content: []collector.ReportEntry{
					{Time: timeutil.MustParseTime(`2000-01-01 01:00:00 +0000`), ID: "some_id", Payload: "some_payload"},
				},
			})

			So(err, ShouldBeNil)

			So(handler.response, ShouldResemble, map[string]interface{}{
				"metadata": map[string]interface{}{
					"is_docker_container":   false,
					"instance_id":           "my-best-uuid",
					"postfix_public_ip":     "127.0.0.2",
					"public_url":            "https://example.com",
					"user_email":            email,
					"user_name":             username,
					"is_using_rsynced_logs": true,
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
			err := (&Dispatcher{
				InstanceID:           "my-best-uuid",
				VersionBuilder:       fakeVersion,
				ReportDestinationURL: s.URL,
				SettingsReader:       m.Reader,
				Auth:                 auth,
				SchedFileReader:      fakeSchedReaderFromContent(schedFileContentForDocker, false, false),
			}).Dispatch(collector.Report{
				Interval: timeutil.TimeInterval{From: timeutil.MustParseTime(`2000-01-01 00:00:00 +0000`), To: timeutil.MustParseTime(`2000-01-01 10:00:00 +0000`)},
				Content: []collector.ReportEntry{
					{Time: timeutil.MustParseTime(`2000-01-01 01:00:00 +0000`), ID: "some_id", Payload: "some_payload"},
				},
			})

			So(err, ShouldBeNil)

			So(handler.response, ShouldResemble, map[string]interface{}{
				"metadata":    map[string]interface{}{"user_email": email, "user_name": username, "instance_id": "my-best-uuid", "is_docker_container": true, "is_using_rsynced_logs": false},
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
			p := postfixversion.NewPublisher(writeRunner.Writer())
			postfixutil.ReadFromTestReader(strings.NewReader("Mar 29 12:55:50 test1 postfix/master[15019]: daemon started -- version 3.4.14, configuration /etc/postfix"), p, 2000, &timeutil.FakeClock{Time: timeutil.MustParseTime(`2000-01-01 15:00:00 +0000`)})
			time.Sleep(100 * time.Millisecond)

			err := (&Dispatcher{
				InstanceID:           "my-best-uuid",
				VersionBuilder:       fakeVersion,
				ReportDestinationURL: s.URL,
				SettingsReader:       m.Reader,
				Auth:                 auth,
				SchedFileReader:      fakeSchedReaderFromContent(schedFileContentForNonDocker, false, false),
			}).Dispatch(collector.Report{
				Interval: timeutil.TimeInterval{From: timeutil.MustParseTime(`2000-01-01 00:00:00 +0000`), To: timeutil.MustParseTime(`2000-01-01 10:00:00 +0000`)},
				Content: []collector.ReportEntry{
					{Time: timeutil.MustParseTime(`2000-01-01 01:00:00 +0000`), ID: "some_id", Payload: "some_payload"},
				},
			})

			So(err, ShouldBeNil)

			So(handler.response, ShouldResemble, map[string]interface{}{
				"metadata":    map[string]interface{}{"user_email": email, "user_name": username, "instance_id": "my-best-uuid", "postfix_version": "3.4.14", "is_docker_container": false, "is_using_rsynced_logs": false},
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
