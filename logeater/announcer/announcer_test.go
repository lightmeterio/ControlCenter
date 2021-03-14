// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package announcer

import (
	"fmt"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"testing"
	"time"
)

type fakeAnnouncer struct {
	startTime time.Time
	p         []Progress
}

func (a *fakeAnnouncer) AnnounceStart(t time.Time) {
	a.startTime = t
}

func (a *fakeAnnouncer) AnnounceProgress(p Progress) {
	a.p = append(a.p, p)
}

type fakeMostRecentTime struct {
	role    string
	times   []time.Time
	index   int
	counter int
}

func (m *fakeMostRecentTime) next() (time.Time, error) {
	m.counter++

	if m.index >= len(m.times) {
		return time.Time{}, fmt.Errorf("Invalid time on %v, index %v", m.role, m.index)
	}

	t := m.times[m.index]

	m.index++

	return t, nil
}

func TestNotifier(t *testing.T) {
	Convey("Notifier can receive more steps than planned", t, func() {
		baseTime := timeutil.MustParseTime(`2000-01-01 00:00:00 +0000`)

		announcer := &fakeAnnouncer{}

		// I plan 4 steps, but will do 7
		notifier := NewNotifier(announcer, 4)
		notifier.Start(baseTime)
		notifier.Step(baseTime.Add(time.Second * 10))
		notifier.Step(baseTime.Add(time.Second * 20))
		notifier.Step(baseTime.Add(time.Second * 30))
		notifier.Step(baseTime.Add(time.Second * 40))
		notifier.Step(baseTime.Add(time.Second * 50))
		notifier.Step(baseTime.Add(time.Second * 60))
		notifier.End(baseTime.Add(time.Second * 70))

		So(announcer.p, ShouldResemble, []Progress{
			{Finished: false, Time: baseTime.Add(time.Second * 10), Progress: 25},
			{Finished: false, Time: baseTime.Add(time.Second * 20), Progress: 50},
			{Finished: false, Time: baseTime.Add(time.Second * 30), Progress: 75},
			{Finished: false, Time: baseTime.Add(time.Second * 40), Progress: 100},
			{Finished: false, Time: baseTime.Add(time.Second * 50), Progress: 100},
			{Finished: false, Time: baseTime.Add(time.Second * 60), Progress: 100},
			{Finished: true, Time: baseTime.Add(time.Second * 70), Progress: 100},
		})

	})
}

func TestSynchronizedAnnouncer(t *testing.T) {
	Convey("Test synchronized announcer", t, func() {
		clock := &timeutil.FakeClock{Time: timeutil.MustParseTime(`2020-10-10 00:00:00 +0000`)}

		// Notice that the import clock is totally independent from the time in the notifications,
		// as it's totally okay to import logs totally contained in the past
		finalAnnouncer := &fakeAnnouncer{}

		// we are importing old logs
		baseTime := timeutil.MustParseTime(`2000-01-01 00:00:00 +0000`)

		// in the application, the primary is a deliverydb.DB{}
		primaryTimes := &fakeMostRecentTime{role: "primary", times: []time.Time{
			baseTime.Add(time.Second * 12),
			baseTime.Add(time.Second * 22),
			baseTime.Add(time.Second * 23),
			baseTime.Add(time.Second * 23),
			baseTime.Add(time.Second * 34),
			baseTime.Add(time.Second * 45),
			// last delivery received, but more logs will come after it, therefore the last time keeps the same
			baseTime.Add(time.Second * 55),
			baseTime.Add(time.Second * 55),
			baseTime.Add(time.Second * 55),
			baseTime.Add(time.Second * 55),
			baseTime.Add(time.Second * 55),
			baseTime.Add(time.Second * 55),
			baseTime.Add(time.Second * 55),
			baseTime.Add(time.Second * 55),
			baseTime.Add(time.Second * 55),
			baseTime.Add(time.Second * 55),
			baseTime.Add(time.Second * 55),
			baseTime.Add(time.Second * 55),
			baseTime.Add(time.Second * 55),
			baseTime.Add(time.Second * 55),
			baseTime.Add(time.Second * 55),
			baseTime.Add(time.Second * 55),
		}}

		// in the application, the secondary is a tracking.Tracker{}
		secondaryTimes := &fakeMostRecentTime{role: "secondary", times: []time.Time{
			baseTime.Add(time.Second * 45),
			baseTime.Add(time.Second * 46),
			baseTime.Add(time.Second * 46),
			baseTime.Add(time.Second * 46),
			baseTime.Add(time.Second * 61),
		}}

		combinedAnnouncer := newSynchronizingAnnouncerWithCustomClock(finalAnnouncer, clock, primaryTimes.next, secondaryTimes.next,
			time.Second*1, time.Second*3, time.Second*6)

		done, cancel := combinedAnnouncer.Run()

		Convey("Skip", func() {
			Skip(combinedAnnouncer)

			So(done(), ShouldBeNil)

			So(finalAnnouncer.startTime.IsZero(), ShouldBeTrue)
			So(finalAnnouncer.p, ShouldResemble, []Progress{
				Progress{Finished: true, Progress: 100, Time: time.Time{}},
			})

			So(primaryTimes.counter, ShouldEqual, 0)
			So(secondaryTimes.counter, ShouldEqual, 0)

			So(clock.Now(), ShouldResemble, timeutil.MustParseTime(`2020-10-10 00:00:00 +0000`))
		})

		Convey("Cancel execution before end", func() {
			notifier := NewNotifier(combinedAnnouncer, 10)

			notifier.Start(baseTime)
			notifier.Step(baseTime.Add(time.Second * 10))
			notifier.Step(baseTime.Add(time.Second * 20))

			// End is not received, and the execution is cancelled
			cancel()
			So(done(), ShouldBeNil)
		})

		Convey("Do not skip", func() {
			notifier := NewNotifier(combinedAnnouncer, 20)

			// The source of progress, normally the logsource
			notifier.Start(baseTime)
			notifier.Step(baseTime.Add(time.Second * 10))
			notifier.Step(baseTime.Add(time.Second * 20))
			notifier.Step(baseTime.Add(time.Second * 30))
			notifier.Step(baseTime.Add(time.Second * 35))
			notifier.Step(baseTime.Add(time.Second * 50))
			notifier.End(baseTime.Add(time.Second * 60))

			So(done(), ShouldBeNil)

			// This looks like an artificial requirement,
			// but it basically ensures t hat the primary and secondary checkers are being used,
			// therefore validating the feature
			So(primaryTimes.counter, ShouldEqual, 11)
			So(secondaryTimes.counter, ShouldEqual, 5)

			So(finalAnnouncer.startTime, ShouldResemble, baseTime)
			So(finalAnnouncer.p, ShouldResemble, []Progress{
				Progress{Finished: false, Progress: 5, Time: baseTime.Add(time.Second * 10)},
				Progress{Finished: false, Progress: 10, Time: baseTime.Add(time.Second * 20)},
				Progress{Finished: false, Progress: 15, Time: baseTime.Add(time.Second * 30)},
				Progress{Finished: false, Progress: 20, Time: baseTime.Add(time.Second * 35)},
				Progress{Finished: false, Progress: 25, Time: baseTime.Add(time.Second * 50)},
				Progress{Finished: true, Progress: 100, Time: baseTime.Add(time.Second * 60)},
			})

			So(clock.Now(), ShouldResemble, timeutil.MustParseTime(`2020-10-10 00:00:09 +0000`))
		})
	})
}
