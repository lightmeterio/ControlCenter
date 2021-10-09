// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package errorutil

import (
	"errors"
	"io"
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
		Convey("There is an error already, and we combine it with the new one", func() {
			origErr := errors.New(`Original one`)
			origErrBefore := origErr
			newErr := errors.New(`New Err`)

			func() {
				defer UpdateErrorFromCall(func() error { return newErr }, &origErr)
			}()

			// we have the original error
			So(errors.Is(origErr, origErrBefore), ShouldBeTrue)

			// and the new one too
			So(errors.Is(origErr, newErr), ShouldBeTrue)
		})

		Convey("Update old error if it's nil", func() {
			var origErr error = nil
			newErr := errors.New(`New Err`)

			func() {
				defer UpdateErrorFromCall(func() error { return newErr }, &origErr)
			}()

			So(origErr, ShouldNotBeNil)
			So(errors.Is(origErr, newErr), ShouldBeTrue)
		})

		Convey("Keep the old log if the new one is nil", func() {
			origErr := errors.New(`Original one`)
			origErrBefore := origErr
			var newErr error = nil

			func() {
				defer UpdateErrorFromCall(func() error { return newErr }, &origErr)
			}()

			So(origErr, ShouldEqual, origErrBefore)
		})

		Convey("Simulate real world typical error handling", func() {
			// this function fails on reading a value, as well as on closing the resource
			err := typicalExampleOfErrorHandling()

			So(errors.Is(err, fakeCloseError), ShouldBeTrue)
			So(errors.Is(err, fakeReadError), ShouldBeTrue)
		})
	})
}

var fakeCloseError = errors.New(`Could not close resource`)
var fakeReadError = errors.New(`Could not read. Or something.`)

type fakeReadCloser struct{}

// Implements io.ReadCloser
func (*fakeReadCloser) Read([]byte) (int, error) {
	return 0, fakeReadError
}

func (*fakeReadCloser) Close() error {
	return fakeCloseError
}

// this function looks like the typical error real world handling flow
func typicalExampleOfErrorHandling() (err error) {
	f := &fakeReadCloser{}

	defer UpdateErrorFromCloser(f, &err)

	// remember this is a different err variable, in its own scope
	if _, err := io.ReadAll(f); err != nil {
		return Wrap(err)
	}

	// Here we would typically use what we've just read

	return nil
}
