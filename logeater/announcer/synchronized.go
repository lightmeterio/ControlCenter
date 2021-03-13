// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package announcer

import (
	"github.com/rs/zerolog/log"
	"gitlab.com/lightmeter/controlcenter/pkg/runner"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"time"
)

type MostRecentLogTimeProvider func() (time.Time, error)

type SynchronizingAnnouncer struct {
	runner.CancelableRunner

	progressChan chan Progress
	announcer    ImportAnnouncer
}

// In Control Center, the primary is deliverydb.DB{} and secondary is tracking.Tracker{}
// the primary expected to never have times higher than the secondary,
// as the tracking is expected to always have log lines more recent (or at least equal) than the times of delivery attempts
func NewSynchronizingAnnouncer(announcer ImportAnnouncer, primary, secondary MostRecentLogTimeProvider) *SynchronizingAnnouncer {
	return newSynchronizingAnnouncerWithCustomClock(announcer, timeutil.RealClock{}, primary, secondary, time.Millisecond*500, time.Second*4, time.Second*3)
}

func (c *SynchronizingAnnouncer) AnnounceStart(t time.Time) {
	c.announcer.AnnounceStart(t)
}

func (c *SynchronizingAnnouncer) AnnounceProgress(p Progress) {
	c.progressChan <- p
}

func waitUntilTimeProvidedIsPastProgressTime(p Progress, checker MostRecentLogTimeProvider,
	timeout time.Duration, clock timeutil.Clock, sleepTime time.Duration) (bool, error) {
	// especial case, when Skip() is called.
	if p.Time.IsZero() {
		return true, nil
	}

	// in the "clock" of the past logs
	var lastReadTime time.Time

	timeOfLastChange := clock.Now()

	stopCondition := func() (shouldStop bool, succeeded bool, err error) {
		t, err := checker()
		if err != nil {
			return true, false, errorutil.Wrap(err)
		}

		log.Debug().Msgf("Read a time (%v) with p = %v and time of last change = (%v)", t, p, timeOfLastChange)

		// found a suitable time
		if t.After(p.Time) {
			log.Debug().Msgf("Yay! Found a time (%v) after progress (%v)!!!!", t, p)
			return true, true, nil
		}

		now := clock.Now()

		// stop in case no changes in the time read have happened in the past `timeout`
		if now.Sub(timeOfLastChange) >= timeout {
			log.Debug().Msgf("Stopping due timeout on now = (%v), lastChange = (%v) and timeout = %v", now, timeOfLastChange, timeout)
			return true, false, nil
		}

		if !lastReadTime.Equal(t) {
			log.Debug().Msgf("Setting lastChange from (%v) to (%v)", timeOfLastChange, now)
			timeOfLastChange = now
		}

		log.Debug().Msgf("Setting last read time from (%v) to (%v) and going to the next iteration", lastReadTime, t)
		lastReadTime = t

		return false, false, nil
	}

	for {
		shouldStop, succeeded, err := stopCondition()
		if err != nil {
			return false, errorutil.Wrap(err)
		}

		if shouldStop {
			return succeeded, nil
		}

		log.Debug().Msgf("Clock sleeping now = (%v)", clock.Now())
		clock.Sleep(sleepTime)
	}
}

func newSynchronizingAnnouncerWithCustomClock(
	announcer ImportAnnouncer,
	clock timeutil.Clock,
	primaryChecker, secondaryChecker MostRecentLogTimeProvider,
	sleepTime time.Duration,
	primaryTimeout time.Duration,
	secondaryTimeout time.Duration,
) *SynchronizingAnnouncer {
	progressChan := make(chan Progress, 1000)

	return &SynchronizingAnnouncer{
		progressChan: progressChan,
		announcer:    announcer,
		CancelableRunner: runner.NewCancelableRunner(func(done runner.DoneChan, cancel runner.CancelChan) {
			go func() {
				for p := range progressChan {
					succeeded, err := waitUntilTimeProvidedIsPastProgressTime(p, primaryChecker, primaryTimeout, clock, sleepTime)
					if err != nil {
						done <- errorutil.Wrap(err)
						return
					}

					// for the last step, we need to ensure that the secondary checker is past the progress time as well
					if p.Finished && !succeeded {
						if _, err := waitUntilTimeProvidedIsPastProgressTime(p, secondaryChecker, secondaryTimeout, clock, sleepTime); err != nil {
							done <- errorutil.Wrap(err)
							return
						}
					}

					log.Debug().Msgf("Announcing progress %v where succeeded is %v", p, succeeded)
					announcer.AnnounceProgress(p)

					if p.Finished {
						break
					}
				}

				done <- nil
			}()

			go func() {
				<-cancel
				close(progressChan)
			}()
		}),
	}
}
