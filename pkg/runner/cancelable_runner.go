// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0

package runner

type CancelChan <-chan struct{}
type DoneChan chan<- error

type CancelableRunner interface {
	Run() (done func() error, cancel func())
}

func NewCancelableRunner(execute func(done DoneChan, cancel CancelChan)) CancelableRunner {
	return &cancelableRunner{
		execute: execute,
	}
}

type cancelableRunner struct {
	execute func(done DoneChan, cancel CancelChan)
}

func (r *cancelableRunner) Run() (func() error, func()) {
	cancel := make(chan struct{})
	done := make(chan error)

	r.execute(done, cancel)

	return func() error {
			return <-done
		}, func() {
			cancel <- struct{}{}
		}
}
