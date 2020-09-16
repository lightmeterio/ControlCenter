package errorutil

import (
	"errors"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func mustSucceed(err error, msg string) func() {
	return func() {
		MustSucceed(err, msg)
	}
}

func TestErrorAssertion(t *testing.T) {
	Convey("Test MustSucceed", t, func() {
		So(mustSucceed(nil, ""), ShouldNotPanic)
		So(mustSucceed(errors.New("Basic Error"), ""), ShouldPanic)
		So(mustSucceed(WrapError(errors.New("Inner Error")), ""), ShouldPanic)
	})
}
