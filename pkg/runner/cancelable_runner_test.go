// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0

package runner

import (
	"errors"
	. "github.com/smartystreets/goconvey/convey"
	"sync/atomic"
	"testing"
)

func TestCancelableRunner(t *testing.T) {
	testErr := errors.New("Something")

	Convey("cancelable runner", t, func() {
		Convey("run", func() {
			var doneCounter int32

			execute := func(done DoneChan, cancel CancelChan) {
				go func() {
					atomic.AddInt32(&doneCounter, 1)
					done <- testErr
				}()

				go func() {
					<-cancel
				}()
			}

			runner := NewCancelableRunner(execute)
			done, cancel := runner.Run()
			cancel()
			err := done()

			So(err, ShouldEqual, testErr)
			So(doneCounter, ShouldEqual, 1)
		})
	})
}
