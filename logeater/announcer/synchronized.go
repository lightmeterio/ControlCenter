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
	finalProgressStepChan chan Progress
	announcer             ImportAnnouncer
}

func NewSynchronizingAnnouncer(announcer ImportAnnouncer, p, s MostRecentLogTimeProvider) *SynchronizingAnnouncer {
	return newSynchronizingAnnouncerWithCustomClock(announcer, timeutil.RealClock{}, p, s, time.Millisecond*500, time.Second*40, time.Second*80)
}

func announceEnd(announcer ImportAnnouncer, p Progress) {
	announcer.AnnounceProgress(p)
}

func (c *SynchronizingAnnouncer) AnnounceStart(t time.Time) {
	c.announcer.AnnounceStart(t)
}

func (c *SynchronizingAnnouncer) AnnounceProgress(p Progress) {
	if p.Finished {
		c.finalProgressStepChan <- p
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
	finalProgressStepChan := make(chan Progress, 1)

	return &SynchronizingAnnouncer{
		progressChan:          progressChan,
		finalProgressStepChan: finalProgressStepChan,
		announcer:             announcer,
		CancelableRunner: runner.NewCancelableRunner(func(done runner.DoneChan, cancel runner.CancelChan) {
			go func() {
				for {
					select {
					case p := <-progressChan:
						log.Debug().Msgf("Received Combined AnnounceProgress: %v", p)

						shouldContinue, err := handleNonFinalProgressStep(
							checkerAndTimeout{primaryChecker, primaryTimeout},
							checkerAndTimeout{secondaryChecker, secondaryTimeout},
							announcer, p, clock, sleepTime, finalProgressStepChan)

						if err != nil {
							log.Debug().Msgf("Combined ImportAnnouce ends with error: %v", err)
							done <- errorutil.Wrap(err)

							return
						}

						if !shouldContinue {
							log.Debug().Msgf("Combined ImportAnnouce ends successfully")
							done <- nil

							return
						}

					case p := <-finalProgressStepChan:
						log.Debug().Msgf("Received Combined Final AnnounceProgress: %v", p)

						if err := handleFinalProgressStep(
							checkerAndTimeout{primaryChecker, primaryTimeout},
							checkerAndTimeout{secondaryChecker, secondaryTimeout},
							announcer, p, clock, sleepTime); err != nil {
							log.Debug().Msgf("Combined ImportAnnouce ends with error: %v", err)
							done <- errorutil.Wrap(err)

							return
						}

						log.Debug().Msgf("Combined ImportAnnouce ends successfully")
						done <- nil

						return
					}
				}
			}()

			go func() {
				<-cancel
				close(progressChan)
				close(finalProgressStepChan)
			}()
		}),
	}
}

func handleNonFinalProgressStep(primary, secondary checkerAndTimeout,
	announcer ImportAnnouncer, p Progress,
	clock timeutil.Clock, sleepTime time.Duration,
	finalProgressStepChan chan Progress,
) (shouldContinue bool, err error) {
	for {
		select {
		case finalStep := <-finalProgressStepChan:
			// Received final step while was processing a non-final one.
			// Just ignore the non final one and use only the final one.
			log.Debug().Msgf("Received last progress step %v while processing progress %v", finalStep, p)

			if err := handleFinalProgressStep(primary, secondary, announcer, finalStep, clock, sleepTime); err != nil {
				return false, errorutil.Wrap(err)
			}

			return false, nil
		default:
			t, err := primary.checker()
			if err != nil {
				return false, errorutil.Wrap(err)
			}

			if t.After(p.Time) {
				log.Debug().Msgf("Combined primary checker unlock importer with progress: %v", p)
				announcer.AnnounceProgress(p)

				return true, nil
			}

			clock.Sleep(sleepTime)
		}
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
	didTimeout, err := announceEndOrTimeout(announcer, p, primary.checker, p.Time.Add(primary.timeout), clock, sleepTime)
	if err != nil {
		return errorutil.Wrap(err)
	}

	if !didTimeout {
		log.Debug().Msgf("Combined primary checker final unlock importer with progress: %v", p)
		return nil
	}

	// tries the secondary, but this one might time out as well
	didTimeout, err = announceEndOrTimeout(announcer, p, secondary.checker, p.Time.Add(secondary.timeout), clock, sleepTime)
	if err != nil {
		return errorutil.Wrap(err)
	}

	if !didTimeout {
		log.Debug().Msgf("Combined secondary checker final unlock importer with progress: %v", p)
		return nil
	}

	log.Debug().Msgf("Combine Give up and just notify progress: %v", p)

	// give up and notify whatever we got
	announceEnd(announcer, p)

	return nil
}

func announceEndOrTimeout(announcer ImportAnnouncer, p Progress,
	checker MostRecentLogTimeProvider, timeout time.Time, clock timeutil.Clock, sleepTime time.Duration) (didTimeout bool, err error) {
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
			return true, nil
		}

		clock.Sleep(sleepTime)
	}
}
