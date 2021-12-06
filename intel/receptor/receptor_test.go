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
	_ "gitlab.com/lightmeter/controlcenter/intel/migrations"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/pkg/dbrunner"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
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

		strAddr := func(s string) string { return s }

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
						Summary: bruteforce.SummaryResult{
							TopIPs: []bruteforce.BlockedIP{
								{Addr: "1.1.1.1", Count: 42},
								{Addr: "2.2.2.2", Count: 35},
							},
							TotalNumber: 77,
						},
					},
				}, nil)

			m.EXPECT().Request(gomock.Any(), Payload{
				InstanceID:       `f5a206d2-6865-4a0a-b04d-423c4ac9d233`,
				LastKnownEventID: strAddr(`8d303a39-44a0-449f-b734-6f1a333ad168`),
				Time:             timeutil.MustParseTime(`2021-11-01 10:00:00 +0000`),
			}).Return(nil, nil)

			clock := &timeutil.FakeClock{Time: timeutil.MustParseTime(`2021-11-01 10:00:10 +0000`)}

			drain(clock, Options{PollInterval: 10 * time.Second, InstanceID: `f5a206d2-6865-4a0a-b04d-423c4ac9d233`})

			events := retrieveAllEvents(db.RoConnPool)

			So(events, ShouldResemble, []Event{
				{ID: `8d303a39-44a0-449f-b734-6f1a333ad168`, Type: "blocked_ips", CreationTime: timeutil.MustParseTime(`2021-11-01 10:00:00 +0000`), BlockedIPs: &BlockedIPs{
					Interval: timeutil.MustParseTimeInterval(`2021-10-01`, `2021-11-01 09:00:00`),
					Summary: bruteforce.SummaryResult{
						TopIPs: []bruteforce.BlockedIP{
							{Addr: "1.1.1.1", Count: 42},
							{Addr: "2.2.2.2", Count: 35},
						},
						TotalNumber: 77,
					},
				}},
			})
		})

		Convey("Two events are generated", func() {
			m.EXPECT().Request(gomock.Any(), Payload{InstanceID: `f5a206d2-6865-4a0a-b04d-423c4ac9d233`}).
				Return(&Event{
					ID:           `8d303a39-44a0-449f-b734-6f1a333ad168`,
					Type:         `blocked_ips`,
					CreationTime: timeutil.MustParseTime(`2021-11-01 10:10:00 +0000`),
					BlockedIPs: &BlockedIPs{
						Interval: timeutil.MustParseTimeInterval(`2021-10-01`, `2021-11-01 09:00:00`),
						Summary: bruteforce.SummaryResult{
							TopIPs: []bruteforce.BlockedIP{
								{Addr: "1.1.1.1", Count: 42},
								{Addr: "2.2.2.2", Count: 35},
							},
							TotalNumber: 77,
						},
					},
				}, nil)

			m.EXPECT().Request(gomock.Any(), Payload{InstanceID: `f5a206d2-6865-4a0a-b04d-423c4ac9d233`, LastKnownEventID: strAddr(`8d303a39-44a0-449f-b734-6f1a333ad168`), Time: timeutil.MustParseTime(`2021-11-01 10:10:00 +0000`)}).
				Return(&Event{
					ID:           `ef086d42-acbe-4f86-94f1-52e8f024fc53`,
					Type:         `action_link`,
					CreationTime: timeutil.MustParseTime(`2021-11-01 10:20:00 +0000`),
					ActionLink: &ActionLink{
						Link:  "http://example.com",
						Label: "Some Link",
					},
				}, nil)

			m.EXPECT().Request(gomock.Any(), Payload{InstanceID: `f5a206d2-6865-4a0a-b04d-423c4ac9d233`, LastKnownEventID: strAddr(`ef086d42-acbe-4f86-94f1-52e8f024fc53`), Time: timeutil.MustParseTime(`2021-11-01 10:20:00 +0000`)}).Return(nil, nil)

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
						Summary: bruteforce.SummaryResult{
							TopIPs: []bruteforce.BlockedIP{
								{Addr: "1.1.1.1", Count: 42},
								{Addr: "2.2.2.2", Count: 35},
							},
							TotalNumber: 77,
						},
					},
				},
				{
					ID:           `ef086d42-acbe-4f86-94f1-52e8f024fc53`,
					Type:         `action_link`,
					CreationTime: timeutil.MustParseTime(`2021-11-01 10:20:00 +0000`),
					ActionLink: &ActionLink{
						Link:  "http://example.com",
						Label: "Some Link",
					},
				},
			})
		})
	})
}
