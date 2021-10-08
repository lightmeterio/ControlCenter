// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package errorutil

import (
	"errors"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func mustSucceed(err error, msg ...string) func() {
	return func() {
		MustSucceed(err, msg...)
	}
}

func TestErrorAssertion(t *testing.T) {
	Convey("Test MustSucceed", t, func() {
		So(mustSucceed(nil, ""), ShouldNotPanic)
		So(mustSucceed(errors.New("Basic Error"), ""), ShouldPanic)
		So(mustSucceed(Wrap(errors.New("Inner Error")), ""), ShouldPanic)
		So(mustSucceed(Wrap(errors.New("Inner Error")), "Hello world"), ShouldPanic)
		So(mustSucceed(nil), ShouldNotPanic)
		So(mustSucceed(nil, "1", "2"), ShouldPanic)
	})
}

func TestDeferredError(t *testing.T) {
	Convey("Test DeferredError", t, func() {
		Convey("There is an error already, so we ignore the new one", func() {
			origErr := errors.New(`Original one`)
			origErrBefore := origErr
			newErr := errors.New(`New Err`)

			func() {
				defer DeferredError(func() error { return newErr }, &origErr)
			}()

			So(origErr, ShouldEqual, origErrBefore)
		})

		Convey("Update old error if it's nil", func() {
			var origErr error = nil
			newErr := errors.New(`New Err`)

			func() {
				defer DeferredError(func() error { return newErr }, &origErr)
			}()

			So(origErr, ShouldNotBeNil)
			So(errors.Is(origErr, newErr), ShouldBeTrue)
		})

		Convey("Keep the old log if the new one is nil", func() {
			origErr := errors.New(`Original one`)
			origErrBefore := origErr
			var newErr error = nil

			func() {
				defer DeferredError(func() error { return newErr }, &origErr)
			}()

			So(origErr, ShouldEqual, origErrBefore)
		})
	})
}
