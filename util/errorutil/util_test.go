// SPDX-FileCopyrightText: 2020,  Lightmeter <hello@lightmeter.io>
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0

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
