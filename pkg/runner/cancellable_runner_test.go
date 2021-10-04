// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package runner

import (
	"errors"
	. "github.com/smartystreets/goconvey/convey"
	"sync"
	"testing"
	"time"
)

func TestCancellableRunner(t *testing.T) {
	testErr := errors.New("Something")

	Convey("cancellable runner", t, func() {
		Convey("run", func() {
			var doneCounter int32

			mutex := sync.Mutex{}

			execute := func(done DoneChan, cancel CancelChan) {
				go func() {
					mutex.Lock()
					doneCounter++
					mutex.Unlock()
					done <- testErr
				}()

				go func() {
					<-cancel
				}()
			}

			runner := NewCancellableRunner(execute)
			done, cancel := Run(runner)
			cancel()
			err := done()

			So(err, ShouldEqual, testErr)
			So(doneCounter, ShouldEqual, 1)
		})
	})
}

func TestMultipleCancellableRunners(t *testing.T) {
	Convey("Multiple Cancellable Runners", t, func() {
		testErr := errors.New("Something")
		var doneCounter int32
		mutex := sync.Mutex{}

		newRunnerWithError := func(err error) CancellableRunner {
			return NewCancellableRunner(func(done DoneChan, cancel CancelChan) {
				go func() {
					mutex.Lock()
					doneCounter++
					mutex.Unlock()
					done <- err
				}()

				go func() {
					<-cancel
				}()
			})
		}

		Convey("All of them succeed", func() {
			done, cancel := Run(newRunnerWithError(nil), newRunnerWithError(nil), newRunnerWithError(nil))

			// to give time for all runners to start
			time.Sleep(50 * time.Millisecond)

			cancel()

			So(done(), ShouldEqual, nil)
			So(doneCounter, ShouldEqual, 3)
		})

		Convey("Some fail", func() {
			// the second one yields an error
			done, cancel := Run(newRunnerWithError(nil), newRunnerWithError(testErr), newRunnerWithError(nil))

			// to give time for all runners to start
			time.Sleep(50 * time.Millisecond)

			cancel()

			So(done(), ShouldEqual, testErr)

			// one runner has failed, but the others might still be running,
			// potentially mutating the counter
			mutex.Lock()
			defer mutex.Unlock()

			So(doneCounter, ShouldEqual, 3)
		})
	})
}
