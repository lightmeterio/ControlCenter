// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package collector

import (
	"context"
	"database/sql"
	"encoding/json"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/intel/core"
	_ "gitlab.com/lightmeter/controlcenter/intel/migrations"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/pkg/runner"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"testing"
	"time"
)

func init() {
	lmsqlite3.Initialize(lmsqlite3.Options{})
}

type fakeReporter struct {
	count    int
	interval time.Duration
	id       string
}

type fakeReportedData struct {
	Info1 int    `json:"info_1"`
	Info2 string `json:"info_2"`
}

func (fakeReporter) Close() error {
	return nil
}

func (f *fakeReporter) ExecutionInterval() time.Duration {
	return f.interval
}

func (f *fakeReporter) ID() string {
	return f.id
}

func (f *fakeReporter) Step(tx *sql.Tx, clock timeutil.Clock) error {
	if err := Collect(tx, clock, f.id, fakeReportedData{
		Info1: f.count,
		Info2: "Saturn",
	}); err != nil {
		return errorutil.Wrap(err)
	}

	f.count++

	return nil
}

type testCollectResult struct {
	time  time.Time
	id    string
	value interface{}
}

func getAllQueuedResults(db *dbconn.PooledPair) ([]testCollectResult, error) {
	conn, release := db.RoConnPool.Acquire()
	defer release()

	r, err := conn.Query(`select time, identifier, value from queued_reports where dispatched_time = 0 order by id`)
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	results := []testCollectResult{}

	for r.Next() {
		var (
			ts   int64
			id   string
			blob string
		)

		err := r.Scan(&ts, &id, &blob)
		if err != nil {
			return nil, errorutil.Wrap(err)
		}

		time := time.Unix(ts, 0).In(time.UTC)

		var value interface{}

		err = json.Unmarshal([]byte(blob), &value)
		if err != nil {
			return nil, errorutil.Wrap(err)
		}

		results = append(results, testCollectResult{time: time, value: value, id: id})
	}

	return results, nil
}

func TestReporters(t *testing.T) {
	Convey("Test Reporters", t, func() {
		db, clear := testutil.TempDBConnectionMigrated(t, "intel-collector")
		defer clear()

		r1 := &fakeReporter{interval: time.Second * 3, id: "fake_1", count: 42}
		r2 := &fakeReporter{interval: time.Second * 5, id: "fake_2", count: 35}

		reporters := Reporters{r1, r2}

		clock := &timeutil.FakeClock{Time: testutil.MustParseTime(`2000-01-01 10:00:00 +0000`)}

		err := db.RwConn.Tx(context.Background(), func(ctx context.Context, tx *sql.Tx) error {
			// nothing executes
			err := reporters.Step(tx, clock)
			So(err, ShouldBeNil)

			// nothing executes
			clock.Sleep(2 * time.Second)
			err = reporters.Step(tx, clock)
			So(err, ShouldBeNil)

			// r1 executes on second 3
			clock.Sleep(1 * time.Second)
			err = reporters.Step(tx, clock)
			So(err, ShouldBeNil)

			// r2 executes on second 5
			clock.Sleep(2 * time.Second)
			err = reporters.Step(tx, clock)
			So(err, ShouldBeNil)

			// r3 executes on second 7
			clock.Sleep(2 * time.Second)
			err = reporters.Step(tx, clock)
			So(err, ShouldBeNil)

			// r1 and r2 execute on second 10
			clock.Sleep(3 * time.Second)
			err = reporters.Step(tx, clock)
			So(err, ShouldBeNil)

			return nil
		})

		So(err, ShouldBeNil)

		results, err := getAllQueuedResults(db)
		So(err, ShouldBeNil)

		So(results, ShouldResemble, []testCollectResult{
			{
				time: testutil.MustParseTime(`2000-01-01 10:00:03 +0000`),
				id:   "fake_1",
				value: map[string]interface{}{
					"info_1": float64(42),
					"info_2": "Saturn",
				},
			},
			{
				time: testutil.MustParseTime(`2000-01-01 10:00:05 +0000`),
				id:   "fake_2",
				value: map[string]interface{}{
					"info_1": float64(35),
					"info_2": "Saturn",
				},
			},
			{
				time: testutil.MustParseTime(`2000-01-01 10:00:07 +0000`),
				id:   "fake_1",
				value: map[string]interface{}{
					"info_1": float64(43),
					"info_2": "Saturn",
				},
			},
			{
				time: testutil.MustParseTime(`2000-01-01 10:00:10 +0000`),
				id:   "fake_1",
				value: map[string]interface{}{
					"info_1": float64(44),
					"info_2": "Saturn",
				},
			},
			{
				time: testutil.MustParseTime(`2000-01-01 10:00:10 +0000`),
				id:   "fake_2",
				value: map[string]interface{}{
					"info_1": float64(36),
					"info_2": "Saturn",
				},
			},
		})
	})
}

func TestDispatcher(t *testing.T) {
	Convey("Test Dispatcher", t, func() {
		db, clear := testutil.TempDBConnectionMigrated(t, "intel-collector")
		defer clear()

		accessor, err := NewAccessor(db.RoConnPool)
		So(err, ShouldBeNil)

		r1 := &fakeReporter{interval: time.Second * 3, id: "fake_1", count: 42}
		r2 := &fakeReporter{interval: time.Second * 5, id: "fake_2", count: 35}

		reporters := Reporters{r1, r2}

		clock := &timeutil.FakeClock{Time: testutil.MustParseTime(`2000-01-01 10:00:00 +0000`)}

		err = db.RwConn.Tx(context.Background(), func(ctx context.Context, tx *sql.Tx) error {
			// nothing executes
			err := reporters.Step(tx, clock)
			So(err, ShouldBeNil)

			// nothing executes
			clock.Sleep(2 * time.Second)
			err = reporters.Step(tx, clock)
			So(err, ShouldBeNil)

			// r1 executes on second 3
			clock.Sleep(1 * time.Second)
			err = reporters.Step(tx, clock)
			So(err, ShouldBeNil)

			// r2 executes on second 5
			clock.Sleep(2 * time.Second)
			err = reporters.Step(tx, clock)
			So(err, ShouldBeNil)

			// r3 executes on second 7
			clock.Sleep(2 * time.Second)
			err = reporters.Step(tx, clock)
			So(err, ShouldBeNil)

			// r1 and r2 execute on second 10
			clock.Sleep(3 * time.Second)
			err = reporters.Step(tx, clock)
			So(err, ShouldBeNil)

			return nil
		})

		So(err, ShouldBeNil)

		// Reports are not dispatched yet
		dispatchedReports, err := accessor.GetDispatchedReports(context.Background())
		So(err, ShouldBeNil)
		So(len(dispatchedReports), ShouldEqual, 0)

		dispatcher := &fakeDispatcher{}

		clock.Sleep(time.Second * 10)

		err = db.RwConn.Tx(context.Background(), func(ctx context.Context, tx *sql.Tx) error {
			err := TryToDispatchReports(tx, clock, dispatcher)
			So(err, ShouldBeNil)
			return nil
		})

		So(err, ShouldBeNil)

		firstReport := Report{
			Interval: timeutil.TimeInterval{From: time.Time{}, To: timeutil.MustParseTime(`2000-01-01 10:00:20 +0000`)},
			Content: []ReportEntry{
				{
					Time: testutil.MustParseTime(`2000-01-01 10:00:03 +0000`),
					ID:   "fake_1",
					Payload: map[string]interface{}{
						"info_1": float64(42),
						"info_2": "Saturn",
					},
				},
				{
					Time: testutil.MustParseTime(`2000-01-01 10:00:05 +0000`),
					ID:   "fake_2",
					Payload: map[string]interface{}{
						"info_1": float64(35),
						"info_2": "Saturn",
					},
				},
				{
					Time: testutil.MustParseTime(`2000-01-01 10:00:07 +0000`),
					ID:   "fake_1",
					Payload: map[string]interface{}{
						"info_1": float64(43),
						"info_2": "Saturn",
					},
				},
				{
					Time: testutil.MustParseTime(`2000-01-01 10:00:10 +0000`),
					ID:   "fake_1",
					Payload: map[string]interface{}{
						"info_1": float64(44),
						"info_2": "Saturn",
					},
				},
				{
					Time: testutil.MustParseTime(`2000-01-01 10:00:10 +0000`),
					ID:   "fake_2",
					Payload: map[string]interface{}{
						"info_1": float64(36),
						"info_2": "Saturn",
					},
				},
			},
		}

		So(len(dispatcher.reports), ShouldEqual, 1)
		So(dispatcher.reports[0], ShouldResemble, firstReport)

		// All results were dispatched, therefore removed from the queue
		{
			results, err := getAllQueuedResults(db)
			So(err, ShouldBeNil)
			So(len(results), ShouldEqual, 0)
		}

		// Now there are dispatched reports
		dispatchedReports, err = accessor.GetDispatchedReports(context.Background())
		So(err, ShouldBeNil)
		So(dispatchedReports, ShouldResemble, []DispatchedReport{
			{
				ID:           5,
				CreationTime: testutil.MustParseTime(`2000-01-01 10:00:10 +0000`),
				DispatchTime: testutil.MustParseTime(`2000-01-01 10:00:20 +0000`),
				Kind:         "fake_2",
				Value: map[string]interface{}{
					"info_1": float64(36),
					"info_2": "Saturn"},
			},
			{
				ID:           4,
				CreationTime: testutil.MustParseTime(`2000-01-01 10:00:10 +0000`),
				DispatchTime: testutil.MustParseTime(`2000-01-01 10:00:20 +0000`),
				Kind:         "fake_1",
				Value: map[string]interface{}{
					"info_1": float64(44),
					"info_2": "Saturn",
				},
			},
			{ID: 3,
				CreationTime: testutil.MustParseTime(`2000-01-01 10:00:07 +0000`),
				DispatchTime: testutil.MustParseTime(`2000-01-01 10:00:20 +0000`),
				Kind:         "fake_1",
				Value: map[string]interface{}{
					"info_1": float64(43),
					"info_2": "Saturn",
				},
			},
			{ID: 2,
				CreationTime: testutil.MustParseTime(`2000-01-01 10:00:05 +0000`),
				DispatchTime: testutil.MustParseTime(`2000-01-01 10:00:20 +0000`),
				Kind:         "fake_2",
				Value: map[string]interface{}{
					"info_1": float64(35),
					"info_2": "Saturn",
				},
			},
			{ID: 1,
				CreationTime: testutil.MustParseTime(`2000-01-01 10:00:03 +0000`),
				DispatchTime: testutil.MustParseTime(`2000-01-01 10:00:20 +0000`),
				Kind:         "fake_1",
				Value: map[string]interface{}{
					"info_1": float64(42),
					"info_2": "Saturn",
				},
			},
		})

		err = db.RwConn.Tx(context.Background(), func(ctx context.Context, tx *sql.Tx) error {
			// r1 and r2 execute on second 12
			clock.Sleep(2 * time.Second)
			err := reporters.Step(tx, clock)
			So(err, ShouldBeNil)

			clock.Sleep(time.Second * 10)
			err = TryToDispatchReports(tx, clock, dispatcher)
			So(err, ShouldBeNil)

			return nil
		})

		// All results were dispatched, therefore removed from the queue
		{
			results, err := getAllQueuedResults(db)
			So(err, ShouldBeNil)
			So(len(results), ShouldEqual, 0)
		}

		// A second report has been dispatched
		secondReport := Report{
			Interval: timeutil.TimeInterval{From: timeutil.MustParseTime(`2000-01-01 10:00:20 +0000`), To: timeutil.MustParseTime(`2000-01-01 10:00:32 +0000`)},
			Content: []ReportEntry{
				{
					Time: testutil.MustParseTime(`2000-01-01 10:00:22 +0000`),
					ID:   "fake_1",
					Payload: map[string]interface{}{
						"info_1": float64(45),
						"info_2": "Saturn",
					},
				},
				{
					Time: testutil.MustParseTime(`2000-01-01 10:00:22 +0000`),
					ID:   "fake_2",
					Payload: map[string]interface{}{
						"info_1": float64(37),
						"info_2": "Saturn",
					},
				},
			},
		}

		So(len(dispatcher.reports), ShouldEqual, 2)
		So(dispatcher.reports[1], ShouldResemble, secondReport)
	})
}

func TestCollectorSteps(t *testing.T) {
	Convey("Test Collector Steps", t, func() {
		db, clear := testutil.TempDBConnectionMigrated(t, "intel-collector")
		defer clear()

		r1 := &fakeReporter{interval: time.Second * 3, id: "fake_1", count: 42}

		reporters := Reporters{r1}

		clock := &timeutil.FakeClock{Time: testutil.MustParseTime(`2000-01-01 10:00:00 +0000`)}

		dispatcher := &fakeDispatcher{}

		err := db.RwConn.Tx(context.Background(), func(ctx context.Context, tx *sql.Tx) error {
			// nothing executes
			err := Step(tx, clock, reporters, dispatcher, time.Second*4)
			So(err, ShouldBeNil)

			// nothing executes
			clock.Sleep(2 * time.Second)
			err = Step(tx, clock, reporters, dispatcher, time.Second*4)
			So(err, ShouldBeNil)

			// r1 executes on second 3
			clock.Sleep(1 * time.Second)
			err = Step(tx, clock, reporters, dispatcher, time.Second*4)
			So(err, ShouldBeNil)

			// a report is dispatched on second 4
			clock.Sleep(1 * time.Second)
			err = Step(tx, clock, reporters, dispatcher, time.Second*4)
			So(err, ShouldBeNil)

			return nil
		})

		So(err, ShouldBeNil)

		So(len(dispatcher.reports), ShouldEqual, 1)
	})
}

func TestCollector(t *testing.T) {
	Convey("Test Collector", t, func() {
		reporters := Reporters{&fakeReporter{interval: 1 * time.Second, id: "fake_1", count: 42}}
		dispatcher := &fakeDispatcher{}

		conn, closeConn := testutil.TempDBConnectionMigrated(t, "intel-collector")
		defer closeConn()

		options := core.Options{CycleInterval: 100 * time.Millisecond, ReportInterval: 2 * time.Second}

		dbRunner := core.NewRunner(conn.RwConn, options)
		dbDone, dbCancel := runner.Run(dbRunner)

		// NOTE: the report times have only precision of seconds only (as they are stored in the database as a int64 timestamp)
		collector, err := New(dbRunner.Actions, options, reporters, dispatcher)
		So(err, ShouldBeNil)

		done, cancel := runner.Run(collector)

		time.Sleep(2100 * time.Millisecond)

		cancel()
		So(done(), ShouldBeNil)

		dbCancel()
		So(dbDone(), ShouldBeNil)

		So(len(dispatcher.reports), ShouldEqual, 1)
		So(len(dispatcher.reports[0].Content), ShouldEqual, 1)
	})
}

func TestRemoveOldDatabaseEntries(t *testing.T) {
	Convey("Remove old entries", t, func() {
		db, clear := testutil.TempDBConnectionMigrated(t, "intel-collector")
		defer clear()

		r1 := &fakeReporter{interval: time.Second * 3, id: "fake_1", count: 42}

		reporters := Reporters{r1}

		clock := &timeutil.FakeClock{Time: testutil.MustParseTime(`2000-01-01 10:00:00 +0000`)}

		dispatcher := &fakeDispatcher{}

		err := db.RwConn.Tx(context.Background(), func(ctx context.Context, tx *sql.Tx) error {
			// nothing executes
			err := Step(tx, clock, reporters, dispatcher, time.Second*4)
			So(err, ShouldBeNil)

			// nothing executes
			clock.Sleep(2 * time.Second)
			err = Step(tx, clock, reporters, dispatcher, time.Second*4)
			So(err, ShouldBeNil)

			// r1 executes on second 3
			clock.Sleep(1 * time.Second)
			err = Step(tx, clock, reporters, dispatcher, time.Second*4)
			So(err, ShouldBeNil)

			// a report is dispatched on second 4
			clock.Sleep(1 * time.Second)
			err = Step(tx, clock, reporters, dispatcher, time.Second*4)
			So(err, ShouldBeNil)

			// We sleep 1h and then r1 AND the report dispatching are executed
			clock.Sleep(1 * time.Hour)
			err = Step(tx, clock, reporters, dispatcher, time.Second*4)
			So(err, ShouldBeNil)

			// We then clean anything older than 1h, removing the first report
			cleaner := core.MakeCleanAction(1 * time.Hour)
			err = cleaner(tx, dbconn.TxPreparedStmts{})
			So(err, ShouldBeNil)

			return nil
		})

		So(err, ShouldBeNil)

		conn, release := db.RoConnPool.Acquire()
		defer release()

		// we keep only one report, the newest one
		var countQueued int
		err = conn.QueryRow(`select count(*) from queued_reports`).Scan(&countQueued)
		So(err, ShouldBeNil)
		So(countQueued, ShouldEqual, 1)

		var countDispatchedTimes int
		err = conn.QueryRow(`select count(*) from dispatch_times`).Scan(&countDispatchedTimes)
		So(err, ShouldBeNil)
		So(countDispatchedTimes, ShouldEqual, 1)

	})
}

type fakeDispatcher struct {
	reports []Report
}

func (f *fakeDispatcher) Dispatch(r Report) error {
	f.reports = append(f.reports, r)
	return nil
}
