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

	progressChan          chan Progress
	finalStepProgressChan chan Progress
	announcer             ImportAnnouncer
}

// In Control Center, the primary is deliverydb.DB{} and secondary is tracking.Tracker{}
// the primary expected to never have times higher than the secondary,
// as the tracking is expected to always have log lines more recent (or at least equal) than the times of delivery attempts
func NewSynchronizingAnnouncer(announcer ImportAnnouncer, primary, secondary MostRecentLogTimeProvider) *SynchronizingAnnouncer {
	return newSynchronizingAnnouncerWithCustomClock(announcer, timeutil.RealClock{}, primary, secondary, time.Millisecond*500, time.Second*10, time.Second*10)
}

func announceEnd(announcer ImportAnnouncer, p Progress) {
	announcer.AnnounceProgress(p)
}

func (c *SynchronizingAnnouncer) AnnounceStart(t time.Time) {
	c.announcer.AnnounceStart(t)
}

func (c *SynchronizingAnnouncer) AnnounceProgress(p Progress) {
	if p.Finished {
		c.finalStepProgressChan <- p
		return
	}

	c.progressChan <- p
}

type checkerAndTimeout struct {
	checker MostRecentLogTimeProvider
	timeout time.Duration
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
	finalStepProgressChan := make(chan Progress, 1)

	return &SynchronizingAnnouncer{
		progressChan:          progressChan,
		finalStepProgressChan: finalStepProgressChan,
		announcer:             announcer,
		CancelableRunner: runner.NewCancelableRunner(func(done runner.DoneChan, cancel runner.CancelChan) {
			go func() {
				// FIXME: this code is a complete mess and needs urgent refactoring!!!
				var finalStepProgress *Progress = nil
				var lastNonFinalStepProgress *Progress = nil

				// FIXME: meh, this is a terrible name
				handleNonFinalHelper := func(p Progress) error {
					if err := handleNonFinalProgressStep(
						primaryChecker, clock.Now().Add(primaryTimeout),
						announcer, p, clock, sleepTime, finalStepProgress); err != nil {
						return errorutil.Wrap(err)
					}

					return nil
				}

				for {
					select {
					case p, ok := <-progressChan:
						if !ok {
							// cancelled. Just leave
							log.Debug().Msgf("Cancelled. Leaving")
							done <- nil

							return
						}

						log.Debug().Msgf("Received Synchronized AnnounceProgress: %v", p)

						lastNonFinalStepProgress = &p

						if err := handleNonFinalHelper(p); err != nil {
							log.Debug().Msgf("Synchronized ImportAnnouce ends with error: %v", err)
							done <- errorutil.Wrap(err)

							return
						}

						log.Debug().Msgf("Synchronized ImportAnnouce Successfully ends (%v)", p)
					case p := <-finalStepProgressChan:
						log.Debug().Msgf("Received the final progress: %v", p)

						finalStepProgress = &p

					drain:
						// drain any previous steps being processed, as they've all already being received!
						for {
							select {
							case p := <-progressChan:
								lastNonFinalStepProgress = &p

								if err := handleNonFinalHelper(p); err != nil {
									log.Debug().Msgf("Draining: Synchronized ImportAnnouce ends with error: %v", err)
									done <- errorutil.Wrap(err)

									return
								}
							default:
								log.Debug().Msgf("Draining: Synchronized could not drain any more progress")
								break drain
							}
						}

						log.Debug().Msgf("Synchronized Done draining it!!")

						if lastNonFinalStepProgress != nil && p.Time == lastNonFinalStepProgress.Time {
							log.Debug().Msgf("Synchronized actually skipping the final step = (%v) vs final = (%v)", lastNonFinalStepProgress.Time, p.Time)
							announceEnd(announcer, p)
							done <- nil

							return
						}

						// And finally, handle final step...
						if err := handleFinalProgressStep(
							checkerAndTimeout{primaryChecker, primaryTimeout},
							checkerAndTimeout{secondaryChecker, secondaryTimeout},
							announcer, p, clock, sleepTime); err != nil {
							log.Debug().Msgf("Draining: Synchronized ImportAnnouce ends with error: %v", err)
							done <- errorutil.Wrap(err)

							return
						}

						done <- nil

						return
					}
				}
			}()

			go func() {
				<-cancel
				close(progressChan)
			}()
		}),
	}
}

func handleNonFinalProgressStep(checker MostRecentLogTimeProvider, timeout time.Time,
	announcer ImportAnnouncer, p Progress,
	clock timeutil.Clock, sleepTime time.Duration,
	finalStepProgress *Progress) error {
	for {
		t, err := checker()
		if err != nil {
			return errorutil.Wrap(err)
		}

		if t.After(p.Time) {
			log.Debug().Msgf("Synchronized primary checker unlock importer with progress: %v", p)
			announcer.AnnounceProgress(p)

			return nil
		}

		// Okay, we time out, and we know the final progress has already been notified. Just give up.
		if finalStepProgress != nil && clock.Now().After(timeout) {
			log.Debug().Msgf("Synchronized gave up trying to process a non final step with p = %v, now = %v and timeout = %v", p, clock.Now(), timeout)

			return nil
		}

		//log.Debug().Msgf("Sleeping on final step = %v, log time = (%v) and now = (%v), timeout = (%v)", p, t, clock.Now(), timeout)
		clock.Sleep(sleepTime)
	}
}

func handleFinalProgressStep(primary, secondary checkerAndTimeout,
	announcer ImportAnnouncer, p Progress,
	clock timeutil.Clock, sleepTime time.Duration) error {
	// Special case, skip announcing
	if p.Time.IsZero() {
		announceEnd(announcer, p)
		return nil
	}

	// finished. Waits for the primary, but falls back to the secondary
	// in case primary does not progress.
	didTimeout, err := announceEndOrTimeout(announcer, p, primary.checker, clock.Now().Add(primary.timeout), clock, sleepTime)
	if err != nil {
		return errorutil.Wrap(err)
	}

	if !didTimeout {
		log.Debug().Msgf("Synchronized primary checker final unlock importer with progress: %v", p)
		return nil
	}

	// tries the secondary, but this one might time out as well
	didTimeout, err = announceEndOrTimeout(announcer, p, secondary.checker, clock.Now().Add(secondary.timeout), clock, sleepTime)
	if err != nil {
		return errorutil.Wrap(err)
	}

	if !didTimeout {
		log.Debug().Msgf("Synchronized secondary checker final unlock importer with progress: %v", p)
		return nil
	}

	log.Debug().Msgf("Synchronize Give up and just notify progress: %v", p)

	// give up and notify whatever we got
	announceEnd(announcer, p)

	return nil
}

func announceEndOrTimeout(announcer ImportAnnouncer, p Progress,
	checker MostRecentLogTimeProvider, timeout time.Time,
	clock timeutil.Clock, sleepTime time.Duration) (didTimeout bool, err error) {
	log.Debug().Msgf("I have progress = %v and my timeout is %v", p, timeout)

	for {
		t, err := checker()
		if err != nil {
			return false, errorutil.Wrap(err)
		}

		if t.After(p.Time) {
			announceEnd(announcer, p)
			return false, nil
		}

		if clock.Now().After(timeout) {
			log.Debug().Msgf("I timed out when now = %v", clock.Now())
			return true, nil
		}

		//log.Debug().Msgf("Sleeping on step = %v, log time = (%v) and now = (%v), timeout = (%v)", p, t, clock.Now(), timeout)
		clock.Sleep(sleepTime)
	}
}
