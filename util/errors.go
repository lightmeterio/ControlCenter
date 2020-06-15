package util

import (
	"fmt"
	"runtime"
)

type Error struct {
	Line     int
	Filename string
	Msg      string
	Err      error
}

func (e *Error) Unwrap() error {
	return e.Err
}

func (e *Error) Error() string {
	if len(e.Msg) > 0 {
		return fmt.Sprintf("%s:%d: \"%s\"", e.Filename, e.Line, e.Msg)
	}

	return fmt.Sprintf("%s:%d", e.Filename, e.Line)
}

// Wrap an error adding more context such as filename and line where wrapping happened
func WrapError(err error, args ...interface{}) *Error {
	_, file, line, ok := runtime.Caller(1)

	if !ok {
		line = 0
		file = `<unknown file>`
	}

	msg := fmt.Sprint(args...)

	return &Error{line, file, msg, err}
}

type Chainable interface {
	Chain() ErrorChain
}

// Return a chain of errors, from top to bottom of the "stack"
func Chain(c Chainable) ErrorChain {
	return c.Chain()
}

func BuildChain(outer, inner error) ErrorChain {
	if err, ok := inner.(Chainable); ok {
		return append(ErrorChain{outer}, err.Chain()...)
	}

	return ErrorChain{outer, inner}
}

func (e *Error) Chain() ErrorChain {
	return BuildChain(e, e.Err)
}

type ErrorChain []error

func (chain ErrorChain) Error() string {
	s := ""
	for _, e := range chain {
		s += "> " + e.Error() + "\n"
	}
	return s
}
