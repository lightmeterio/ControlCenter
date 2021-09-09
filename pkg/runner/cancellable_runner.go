// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package runner

import (
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"reflect"
)

type CancelChan <-chan struct{}
type DoneChan chan<- error

type CancellableRunner interface {
	Run() (done func() error, cancel func())
	self() *cancellableRunner
}

func NewCancellableRunner(execute func(done DoneChan, cancel CancelChan)) CancellableRunner {
	return &cancellableRunner{
		execute: execute,
	}
}

type cancellableRunner struct {
	execute func(done DoneChan, cancel CancelChan)
}

func (r *cancellableRunner) Run() (func() error, func()) {
	return Run(r)
}

func (r *cancellableRunner) self() *cancellableRunner {
	return r
}

// Run runs all the runners (duh!) and when `done` is called,
// it returns only the first error generated, or nil, in case no errors happened.
func Run(runners ...CancellableRunner) (done func() error, cancel func()) {
	cancelChannels := make([]chan struct{}, 0, len(runners))
	branches := make([]reflect.SelectCase, 0, len(runners))

	for _, runner := range runners {
		cancelChan := make(chan struct{}, 1)
		doneChan := make(chan error)

		cancelChannels = append(cancelChannels, cancelChan)

		branches = append(branches, reflect.SelectCase{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(doneChan),
		})

		runner.self().execute(doneChan, cancelChan)
	}

	return func() error {
			for len(branches) > 0 {
				index, recv, _ := reflect.Select(branches)
				errInterface := recv.Interface()

				if errInterface != nil {
					return errInterface.(error)
				}

				// remove index, not caring about the order
				branches[index] = branches[len(branches)-1]
				branches = branches[:len(branches)-1]
			}

			return nil
		}, func() {
			for _, c := range cancelChannels {
				c <- struct{}{}
			}
		}
}

type CombinedCancellableRunners struct {
	runners []CancellableRunner
	runner  *cancellableRunner
}

func NewCombinedCancellableRunners(runners ...CancellableRunner) CancellableRunner {
	return &CombinedCancellableRunners{
		runners: runners,
		runner: &cancellableRunner{
			execute: func(done DoneChan, cancel CancelChan) {
				doneAll, cancelAll := Run(runners...)

				go func() {
					<-cancel
					cancelAll()
				}()

				go func() {
					if err := doneAll(); err != nil {
						done <- errorutil.Wrap(err)
						return
					}

					done <- nil
				}()
			},
		},
	}
}

func (r *CombinedCancellableRunners) Run() (func() error, func()) {
	return Run(r)
}

func (r *CombinedCancellableRunners) self() *cancellableRunner {
	return r.runner
}
