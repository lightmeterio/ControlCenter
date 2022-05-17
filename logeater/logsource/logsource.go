// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package logsource

import (
	"gitlab.com/lightmeter/controlcenter/pkg/postfix"
	"gitlab.com/lightmeter/controlcenter/pkg/runner"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
)

type Source interface {
	PublishLogs(postfix.Publisher) error
}

type ComposedSource []Source

func (c ComposedSource) PublishLogs(p postfix.Publisher) error {
	runners := []runner.CancellableRunner{}

	for _, s := range c {
		// we need to create a copy of the source (s), otherwise the goroutine will capture only the last value.
		// Damn passing by reference, Go!
		func(s Source) {
			runners = append(runners, runner.NewCancellableRunner(func(done runner.DoneChan, cancel runner.CancelChan) {
				go func() {
					err := s.PublishLogs(p)
					if err != nil {
						done <- errorutil.Wrap(err)
						return
					}

					done <- nil
				}()
			}))
		}(s)
	}

	done, _ := runner.Run(runner.NewCombinedCancellableRunners(runners...))

	if err := done(); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}
