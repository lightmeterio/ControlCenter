package util

import (
	"log"
	"path"
	"runtime"
	"strings"
	"testing"

	"errors"

	. "github.com/smartystreets/goconvey/convey"
)

func getLine() int {
	_, _, line, ok := runtime.Caller(1)

	if !ok {
		panic("Could not get line number on test")
	}

	return line
}

func TestErrorWrapping(t *testing.T) {
	Convey("Empty message", t, func() {
		err := errors.New("Boom")
		w, line := WrapError(err), getLine()
		So(errors.Is(w, err), ShouldBeTrue)
		So(path.Base(w.Filename), ShouldEqual, "errors_test.go")
		// the line where w was created. This assertion will need updating if code moves around
		So(w.Line, ShouldEqual, line)
		So(w.Msg, ShouldEqual, "")
	})

	Convey("Non empty message", t, func() {
		err := errors.New("Boom")
		w, line := WrapError(err, "This is the ", "Answer: ", 42), getLine()
		So(errors.Is(w, err), ShouldBeTrue)
		So(path.Base(w.Filename), ShouldEqual, "errors_test.go")
		// the line where w was created. This assertion will need updating if code moves around
		So(w.Line, ShouldEqual, line)
		So(w.Msg, ShouldEqual, "This is the Answer: 42")
	})

	Convey("Errors chain", t, func() {
		e1 := errors.New("e1")
		e2 := WrapError(e1, "wrapping e1")
		e3 := WrapError(e2)
		e4 := WrapError(e3)

		So(e2.Chain(), ShouldResemble, ErrorChain{e2, e1})
		So(e4.Chain(), ShouldResemble, ErrorChain{e4, e3, e2, e1})
		So(errors.Is(e4, e1), ShouldBeTrue)

		countLines := func(c ErrorChain) int {
			msg := strings.Trim(c.Error(), "\n")
			log.Println("{", msg, "}")
			return len(strings.Split(msg, "\n"))
		}

		So(countLines(e2.Chain()), ShouldEqual, 2)
		So(countLines(e3.Chain()), ShouldEqual, 3)
		So(countLines(e4.Chain()), ShouldEqual, 4)
	})
}
