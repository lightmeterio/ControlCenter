// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package receptor

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/intel/bruteforce"
	"gitlab.com/lightmeter/controlcenter/intel/core"
	_ "gitlab.com/lightmeter/controlcenter/intel/migrations"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/pkg/dbrunner"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func init() {
	lmsqlite3.Initialize(lmsqlite3.Options{})
}

func retrieveAllEvents(pool *dbconn.RoPool) []Event {
	conn, release := pool.Acquire()
	defer release()

	rows, err := conn.Query(`select content from events order by id`)
	So(err, ShouldBeNil)

	events := []Event{}

	for rows.Next() {
		var (
			raw   string
			event Event
		)

		err := rows.Scan(&raw)
		So(err, ShouldBeNil)
		err = json.Unmarshal([]byte(raw), &event)
		So(err, ShouldBeNil)

		events = append(events, event)
	}

	So(rows.Err(), ShouldBeNil)

	return events
}

func TestReceptor(t *testing.T) {
	Convey("Test Receptor", t, func() {
		db, clear := testutil.TempDBConnectionMigrated(t, "intel-collector")
		defer clear()

		ctrl := gomock.NewController(t)
		m := NewMockRequester(ctrl)

		defer ctrl.Finish()

		drain := func(clock timeutil.Clock, options Options) {
			actions := make(chan dbrunner.Action, 1024)

			err := DrainEvents(actions, options, m, clock)
			So(err, ShouldBeNil)

			close(actions)

			db.RwConn.Tx(context.Background(), func(_ context.Context, tx *sql.Tx) error {
				for action := range actions {
					err := action(tx, dbconn.TxPreparedStmts{})
					So(err, ShouldBeNil)
				}

				return nil
			})
		}

		Convey("Start with no events available", func() {
			m.EXPECT().Request(gomock.Any(), Payload{InstanceID: `f5a206d2-6865-4a0a-b04d-423c4ac9d233`}).Return(nil, nil)

			clock := &timeutil.FakeClock{Time: timeutil.MustParseTime(`2021-11-01 10:00:10 +0000`)}

			drain(clock, Options{PollInterval: 10 * time.Second, InstanceID: `f5a206d2-6865-4a0a-b04d-423c4ac9d233`})

			events := retrieveAllEvents(db.RoConnPool)

			So(events, ShouldResemble, []Event{})
		})

		Convey("One event is generated in total", func() {
			m.EXPECT().Request(gomock.Any(), Payload{InstanceID: `f5a206d2-6865-4a0a-b04d-423c4ac9d233`}).
				Return(&Event{
					ID:           `8d303a39-44a0-449f-b734-6f1a333ad168`,
					Type:         `blocked_ips`,
					CreationTime: timeutil.MustParseTime(`2021-11-01 10:00:00 +0000`),
					BlockedIPs: &BlockedIPs{
						Interval: timeutil.MustParseTimeInterval(`2021-10-01`, `2021-11-01 09:00:00`),
						List: []BlockedIP{
							{Address: "1.1.1.1", Count: 42},
							{Address: "2.2.2.2", Count: 35},
						},
					},
				}, nil)

			m.EXPECT().Request(gomock.Any(), Payload{
				InstanceID:       `f5a206d2-6865-4a0a-b04d-423c4ac9d233`,
				LastKnownEventID: string(`8d303a39-44a0-449f-b734-6f1a333ad168`),
				CreationTime:     timeutil.MustParseTime(`2021-11-01 10:00:00 +0000`),
			}).Return(nil, nil)

			clock := &timeutil.FakeClock{Time: timeutil.MustParseTime(`2021-11-01 10:00:10 +0000`)}

			drain(clock, Options{PollInterval: 10 * time.Second, InstanceID: `f5a206d2-6865-4a0a-b04d-423c4ac9d233`})

			events := retrieveAllEvents(db.RoConnPool)

			So(events, ShouldResemble, []Event{
				{ID: `8d303a39-44a0-449f-b734-6f1a333ad168`, Type: "blocked_ips", CreationTime: timeutil.MustParseTime(`2021-11-01 10:00:00 +0000`), BlockedIPs: &BlockedIPs{
					Interval: timeutil.MustParseTimeInterval(`2021-10-01`, `2021-11-01 09:00:00`),
					List: []BlockedIP{
						{Address: "1.1.1.1", Count: 42},
						{Address: "2.2.2.2", Count: 35},
					},
				}},
			})
		})

		Convey("Three events are generated, and the oldest one deleted", func() {
			m.EXPECT().Request(gomock.Any(), Payload{InstanceID: `f5a206d2-6865-4a0a-b04d-423c4ac9d233`}).
				Return(&Event{
					ID:           `8d303a39-44a0-449f-b734-6f1a333ad168`,
					Type:         `blocked_ips`,
					CreationTime: timeutil.MustParseTime(`2021-11-01 10:10:00 +0000`),
					BlockedIPs: &BlockedIPs{
						Interval: timeutil.MustParseTimeInterval(`2021-10-01`, `2021-11-01 09:00:00`),
						List: []BlockedIP{
							{Address: "1.1.1.1", Count: 42},
							{Address: "2.2.2.2", Count: 35},
						},
					},
				}, nil)

			m.EXPECT().Request(gomock.Any(), Payload{InstanceID: `f5a206d2-6865-4a0a-b04d-423c4ac9d233`, LastKnownEventID: string(`8d303a39-44a0-449f-b734-6f1a333ad168`), CreationTime: timeutil.MustParseTime(`2021-11-01 10:10:00 +0000`)}).
				Return(&Event{
					ID:           `ef086d42-acbe-4f86-94f1-52e8f024fc53`,
					Type:         `action_link`,
					CreationTime: timeutil.MustParseTime(`2021-11-01 10:20:00 +0000`),
					MessageNotification: &MessageNotification{
						Severity: "WARN",
						Title:    "Some Msg",
						Message:  "THis is a message",
						ActionLink: &ActionLink{
							Link:  "http://example.com",
							Label: "Some Link",
						},
					},
				}, nil)

			m.EXPECT().Request(gomock.Any(), Payload{InstanceID: `f5a206d2-6865-4a0a-b04d-423c4ac9d233`, LastKnownEventID: string(`ef086d42-acbe-4f86-94f1-52e8f024fc53`), CreationTime: timeutil.MustParseTime(`2021-11-01 10:20:00 +0000`)}).Return(nil, nil)

			// after one day, a new event is generated
			m.EXPECT().Request(gomock.Any(), Payload{InstanceID: `f5a206d2-6865-4a0a-b04d-423c4ac9d233`, LastKnownEventID: string(`ef086d42-acbe-4f86-94f1-52e8f024fc53`), CreationTime: timeutil.MustParseTime(`2021-11-01 10:20:00 +0000`)}).
				Return(&Event{
					ID:           `f930eeba-415e-49f0-849d-f238009d670f`,
					Type:         `action_link`,
					CreationTime: timeutil.MustParseTime(`2021-11-02 10:20:00 +0000`),
					MessageNotification: &MessageNotification{
						Severity: "WARN",
						Title:    "Some Msg",
						Message:  "THis is a message",
						ActionLink: &ActionLink{
							Link:  "http://second-link.com",
							Label: "Second Link",
						},
					},
				}, nil)

			m.EXPECT().Request(gomock.Any(), Payload{InstanceID: `f5a206d2-6865-4a0a-b04d-423c4ac9d233`, LastKnownEventID: `f930eeba-415e-49f0-849d-f238009d670f`, CreationTime: timeutil.MustParseTime(`2021-11-02 10:20:00 +0000`)}).Return(nil, nil)

			clock := &timeutil.FakeClock{Time: timeutil.MustParseTime(`2021-11-01 10:00:10 +0000`)}

			drain(clock, Options{PollInterval: 10 * time.Second, InstanceID: `f5a206d2-6865-4a0a-b04d-423c4ac9d233`})

			clock.Sleep(time.Hour * 24)

			drain(clock, Options{PollInterval: 10 * time.Second, InstanceID: `f5a206d2-6865-4a0a-b04d-423c4ac9d233`})

			So(db.RwConn.Tx(context.Background(), func(_ context.Context, tx *sql.Tx) error {
				// delete the first event, which is old
				return core.MakeCleanAction(12*time.Hour)(tx, dbconn.TxPreparedStmts{})
			}), ShouldBeNil)

			events := retrieveAllEvents(db.RoConnPool)

			// only the last event remains
			So(events, ShouldResemble, []Event{
				{
					ID:           `f930eeba-415e-49f0-849d-f238009d670f`,
					Type:         `action_link`,
					CreationTime: timeutil.MustParseTime(`2021-11-02 10:20:00 +0000`),
					MessageNotification: &MessageNotification{
						Severity: "WARN",
						Title:    "Some Msg",
						Message:  "THis is a message",
						ActionLink: &ActionLink{
							Link:  "http://second-link.com",
							Label: "Second Link",
						},
					},
				},
			})
		})
	})
}

func TestHTTPReceptor(t *testing.T) {
	Convey("Test HTTP Receptor", t, func() {
		db, clear := testutil.TempDBConnectionMigrated(t, "intel-collector")
		defer clear()

		ctrl := gomock.NewController(t)
		m := NewMockRequester(ctrl)

		defer ctrl.Finish()

		requestTimeIndex := 0
		var requestTimes []time.Time

		pushRequestTime := func(time time.Time) {
			requestTimes = append(requestTimes, time)
		}

		nextRequestTime := func() time.Time {
			defer func() { requestTimeIndex++ }()
			return requestTimes[requestTimeIndex]
		}

		// this function mimics the actual events fetching endpoint from netint
		endpoint := func(w http.ResponseWriter, r *http.Request) {
			instanceID := r.FormValue("instance-id")
			eventID := r.FormValue("event-id")

			event, err := m.Request(r.Context(), Payload{CreationTime: nextRequestTime(), InstanceID: instanceID, LastKnownEventID: eventID})
			if event == nil {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			json.NewEncoder(w).Encode(event)
		}

		s := httptest.NewServer(http.HandlerFunc(endpoint))

		requester := HTTPRequester{URL: s.URL, Timeout: 10 * time.Millisecond}

		drain := func(clock timeutil.Clock, options Options) {
			actions := make(chan dbrunner.Action, 1024)

			err := DrainEvents(actions, options, &requester, clock)
			So(err, ShouldBeNil)

			close(actions)

			db.RwConn.Tx(context.Background(), func(_ context.Context, tx *sql.Tx) error {
				for action := range actions {
					err := action(tx, dbconn.TxPreparedStmts{})
					So(err, ShouldBeNil)
				}

				return nil
			})
		}

		Convey("Two events are generated", func() {
			pushRequestTime(time.Time{})

			m.EXPECT().Request(gomock.Any(), Payload{InstanceID: `f5a206d2-6865-4a0a-b04d-423c4ac9d233`}).
				Return(&Event{
					ID:           `8d303a39-44a0-449f-b734-6f1a333ad168`,
					Type:         `blocked_ips`,
					CreationTime: timeutil.MustParseTime(`2021-11-01 10:10:00 +0000`),
					BlockedIPs: &BlockedIPs{
						Interval: timeutil.MustParseTimeInterval(`2021-10-01`, `2021-11-01 09:00:00`),
						List: []BlockedIP{
							{Address: "1.1.1.1", Count: 42},
							{Address: "2.2.2.2", Count: 35},
						},
					},
				}, nil)

			pushRequestTime(timeutil.MustParseTime(`2021-11-01 10:10:00 +0000`))

			m.EXPECT().Request(gomock.Any(), Payload{InstanceID: `f5a206d2-6865-4a0a-b04d-423c4ac9d233`, LastKnownEventID: string(`8d303a39-44a0-449f-b734-6f1a333ad168`), CreationTime: timeutil.MustParseTime(`2021-11-01 10:10:00 +0000`)}).
				Return(&Event{
					ID:           `ef086d42-acbe-4f86-94f1-52e8f024fc53`,
					Type:         `action_link`,
					CreationTime: timeutil.MustParseTime(`2021-11-01 10:20:00 +0000`),
					MessageNotification: &MessageNotification{
						Severity: "WARN",
						Title:    "Some Msg",
						Message:  "THis is a message",
						ActionLink: &ActionLink{
							Link:  "http://example.com",
							Label: "Some Link",
						},
					},
				}, nil)

			pushRequestTime(timeutil.MustParseTime(`2021-11-01 10:20:00 +0000`))

			m.EXPECT().Request(gomock.Any(), Payload{InstanceID: `f5a206d2-6865-4a0a-b04d-423c4ac9d233`, LastKnownEventID: string(`ef086d42-acbe-4f86-94f1-52e8f024fc53`), CreationTime: timeutil.MustParseTime(`2021-11-01 10:20:00 +0000`)}).Return(nil, nil)

			clock := &timeutil.FakeClock{Time: timeutil.MustParseTime(`2021-11-01 10:00:10 +0000`)}

			drain(clock, Options{PollInterval: 10 * time.Second, InstanceID: `f5a206d2-6865-4a0a-b04d-423c4ac9d233`})

			events := retrieveAllEvents(db.RoConnPool)

			So(events, ShouldResemble, []Event{
				{
					ID:           `8d303a39-44a0-449f-b734-6f1a333ad168`,
					Type:         `blocked_ips`,
					CreationTime: timeutil.MustParseTime(`2021-11-01 10:10:00 +0000`),
					BlockedIPs: &BlockedIPs{
						Interval: timeutil.MustParseTimeInterval(`2021-10-01`, `2021-11-01 09:00:00`),
						List: []BlockedIP{
							{Address: "1.1.1.1", Count: 42},
							{Address: "2.2.2.2", Count: 35},
						},
					},
				},
				{
					ID:           `ef086d42-acbe-4f86-94f1-52e8f024fc53`,
					Type:         `action_link`,
					CreationTime: timeutil.MustParseTime(`2021-11-01 10:20:00 +0000`),
					MessageNotification: &MessageNotification{
						Severity: "WARN",
						Title:    "Some Msg",
						Message:  "THis is a message",
						ActionLink: &ActionLink{
							Link:  "http://example.com",
							Label: "Some Link",
						},
					},
				},
			})
		})
	})
}

func TestBruteforceChecker(t *testing.T) {
	Convey("Test bruteforce checker", t, func() {
		db, clear := testutil.TempDBConnectionMigrated(t, "intel-collector")
		defer clear()

		ctrl := gomock.NewController(t)
		m := NewMockRequester(ctrl)

		defer ctrl.Finish()

		drain := func(clock timeutil.Clock, options Options) {
			actions := make(chan dbrunner.Action, 1024)

			err := DrainEvents(actions, options, m, clock)
			So(err, ShouldBeNil)

			close(actions)

			db.RwConn.Tx(context.Background(), func(_ context.Context, tx *sql.Tx) error {
				for action := range actions {
					err := action(tx, dbconn.TxPreparedStmts{})
					So(err, ShouldBeNil)
				}

				return nil
			})
		}

		withActions := func(clock timeutil.Clock, f func(actions dbrunner.Actions, clock timeutil.Clock) error) {
			actions := make(chan dbrunner.Action, 1024)

			So(f(actions, clock), ShouldBeNil)

			close(actions)

			db.RwConn.Tx(context.Background(), func(_ context.Context, tx *sql.Tx) error {
				for action := range actions {
					err := action(tx, dbconn.TxPreparedStmts{})
					So(err, ShouldBeNil)
				}

				return nil
			})
		}

		Convey("No new results", func() {
			m.EXPECT().Request(gomock.Any(), Payload{InstanceID: `f5a206d2-6865-4a0a-b04d-423c4ac9d233`}).Return(nil, nil)

			clock := &timeutil.FakeClock{Time: timeutil.MustParseTime(`2021-11-01 10:00:00 +0000`)}

			drain(clock, Options{PollInterval: 10 * time.Millisecond, InstanceID: `f5a206d2-6865-4a0a-b04d-423c4ac9d233`})

			So(func() {
				withActions(clock, func(actions dbrunner.Actions, clock timeutil.Clock) error {
					checker := &dbBruteForceChecker{pool: db.RoConnPool, actions: actions, listMaxSize: 100}

					return checker.Step(clock.Now(), func(r bruteforce.SummaryResult) error {
						panic("Should not be called!")
					})
				})
			}, ShouldNotPanic)
		})

		Convey("One new event generates a new result", func() {
			m.EXPECT().Request(gomock.Any(), Payload{InstanceID: `f5a206d2-6865-4a0a-b04d-423c4ac9d233`}).
				Return(&Event{
					ID:           `8d303a39-44a0-449f-b734-6f1a333ad168`,
					Type:         `blocked_ips`,
					CreationTime: timeutil.MustParseTime(`2021-11-01 10:00:00 +0000`),
					BlockedIPs: &BlockedIPs{
						Interval: timeutil.MustParseTimeInterval(`2021-10-01`, `2021-11-01 09:00:00`),
						List: []BlockedIP{
							{Address: "1.1.1.1", Count: 42},
							{Address: "3.3.3.3", Count: 10},
							{Address: "2.2.2.2", Count: 35},
						},
					},
				}, nil)

			m.EXPECT().Request(gomock.Any(), Payload{InstanceID: `f5a206d2-6865-4a0a-b04d-423c4ac9d233`, LastKnownEventID: `8d303a39-44a0-449f-b734-6f1a333ad168`, CreationTime: timeutil.MustParseTime(`2021-11-01 10:00:00 +0000`)}).Return(nil, nil)

			clock := &timeutil.FakeClock{Time: timeutil.MustParseTime(`2021-11-01 10:00:00 +0000`)}

			drain(clock, Options{PollInterval: 10 * time.Millisecond, InstanceID: `f5a206d2-6865-4a0a-b04d-423c4ac9d233`})

			var results []*bruteforce.SummaryResult

			// we are checking a bit in the future
			clock.Sleep(time.Minute * 2)

			withActions(clock, func(actions dbrunner.Actions, clock timeutil.Clock) error {
				checker := &dbBruteForceChecker{pool: db.RoConnPool, actions: actions, listMaxSize: 2}

				return checker.Step(clock.Now(), func(r bruteforce.SummaryResult) error {
					results = append(results, &r)
					return nil
				})
			})

			clock.Sleep(time.Minute * 2)

			// a new result should not be generated as no new event was created
			withActions(clock, func(actions dbrunner.Actions, clock timeutil.Clock) error {
				checker := &dbBruteForceChecker{pool: db.RoConnPool, actions: actions, listMaxSize: 2}

				return checker.Step(clock.Now(), func(r bruteforce.SummaryResult) error {
					results = append(results, &r)
					return nil
				})
			})

			So(results, ShouldResemble, []*bruteforce.SummaryResult{
				&bruteforce.SummaryResult{
					TopIPs: []bruteforce.BlockedIP{
						{Address: "1.1.1.1", Count: 42},
						{Address: "2.2.2.2", Count: 35},
					},
					TotalNumber: 87,
				},
			})
		})

		Convey("Two results", func() {
			m.EXPECT().Request(gomock.Any(), Payload{InstanceID: `f5a206d2-6865-4a0a-b04d-423c4ac9d233`}).
				Return(&Event{
					ID:           `8d303a39-44a0-449f-b734-6f1a333ad168`,
					Type:         `blocked_ips`,
					CreationTime: timeutil.MustParseTime(`2021-11-01 10:00:00 +0000`),
					BlockedIPs: &BlockedIPs{
						Interval: timeutil.MustParseTimeInterval(`2021-10-01`, `2021-11-01 09:00:00`),
						List: []BlockedIP{
							{Address: "2.2.2.2", Count: 35},
							{Address: "1.1.1.1", Count: 42},
							{Address: "3.3.3.3", Count: 10},
							{Address: "4.4.4.4", Count: 145},
						},
					},
				}, nil)

			// initially nothing there
			m.EXPECT().Request(gomock.Any(), Payload{InstanceID: `f5a206d2-6865-4a0a-b04d-423c4ac9d233`, LastKnownEventID: `8d303a39-44a0-449f-b734-6f1a333ad168`, CreationTime: timeutil.MustParseTime(`2021-11-01 10:00:00 +0000`)}).Return(nil, nil)

			// but after 1 day we get a new event
			m.EXPECT().Request(gomock.Any(), Payload{InstanceID: `f5a206d2-6865-4a0a-b04d-423c4ac9d233`, LastKnownEventID: `8d303a39-44a0-449f-b734-6f1a333ad168`, CreationTime: timeutil.MustParseTime(`2021-11-01 10:00:00 +0000`)}).
				Return(&Event{
					ID:           `ef086d42-acbe-4f86-94f1-52e8f024fc53`,
					Type:         `blocked_ips`,
					CreationTime: timeutil.MustParseTime(`2021-11-02 10:00:00 +0000`),
					BlockedIPs: &BlockedIPs{
						Interval: timeutil.MustParseTimeInterval(`2021-10-02`, `2021-11-02 09:00:00`),
						List: []BlockedIP{
							{Address: "1.1.1.1", Count: 42},
							{Address: "2.2.2.2", Count: 35},
							{Address: "5.5.5.5", Count: 5},
							{Address: "1.2.3.4", Count: 10},
							{Address: "4.3.2.1", Count: 10},
						},
					},
				}, nil)

			m.EXPECT().Request(gomock.Any(), Payload{InstanceID: `f5a206d2-6865-4a0a-b04d-423c4ac9d233`, LastKnownEventID: `ef086d42-acbe-4f86-94f1-52e8f024fc53`, CreationTime: timeutil.MustParseTime(`2021-11-02 10:00:00 +0000`)}).Return(nil, nil)

			clock := &timeutil.FakeClock{Time: timeutil.MustParseTime(`2021-11-01 10:00:00 +0000`)}

			var results []*bruteforce.SummaryResult

			drain(clock, Options{PollInterval: 10 * time.Second, InstanceID: `f5a206d2-6865-4a0a-b04d-423c4ac9d233`})

			// we are checking a bit in the future
			clock.Sleep(time.Minute * 2)

			withActions(clock, func(actions dbrunner.Actions, clock timeutil.Clock) error {
				checker := &dbBruteForceChecker{pool: db.RoConnPool, actions: actions, listMaxSize: 2}

				return checker.Step(clock.Now(), func(r bruteforce.SummaryResult) error {
					results = append(results, &r)
					return nil
				})
			})

			clock.Sleep(24 * time.Hour)

			drain(clock, Options{PollInterval: 10 * time.Second, InstanceID: `f5a206d2-6865-4a0a-b04d-423c4ac9d233`})

			clock.Sleep(time.Minute * 2)

			withActions(clock, func(actions dbrunner.Actions, clock timeutil.Clock) error {
				checker := &dbBruteForceChecker{pool: db.RoConnPool, actions: actions, listMaxSize: 2}

				return checker.Step(clock.Now(), func(r bruteforce.SummaryResult) error {
					results = append(results, &r)
					return nil
				})
			})

			So(results, ShouldResemble, []*bruteforce.SummaryResult{
				&bruteforce.SummaryResult{
					TopIPs: []bruteforce.BlockedIP{
						{Address: "4.4.4.4", Count: 145},
						{Address: "1.1.1.1", Count: 42},
					},
					TotalNumber: 232,
				},
				&bruteforce.SummaryResult{
					TopIPs: []bruteforce.BlockedIP{
						{Address: "1.1.1.1", Count: 42},
						{Address: "2.2.2.2", Count: 35},
					},
					TotalNumber: 102,
				},
			})
		})
	})
}
