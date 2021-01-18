package errorutil

import (
	"errors"
	"fmt"
	"reflect"
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
		return e.Msg
	}

	return e.Err.Error()
}

func wrap(skip int, err error, args ...interface{}) *Error {
	_, file, line, ok := runtime.Caller(skip)

	if !ok {
		line = 0
		file = `<unknown file>`
	}

	msg := fmt.Sprint(args...)

	return &Error{line, file, msg, err}
}

// Wrap an error adding more context such as filename and line where wrapping happened
func Wrap(err error, args ...interface{}) *Error {
	return wrap(2, err, args...)
}

// MaybeWrap wraps an error if it exists. See Wrap() for more info
func MaybeWrap(err error, args ...interface{}) *Error {
	if err == nil {
		return nil
	}

	return wrap(2, err, args...)
}

// Given an error, tries to unwrap it recursively until finds a "trivial" error
// which cannot be unwrapped anymore.
// It differs from errors.Unwrap() on returning the error itself
// in case it cannot be unwrapped, instead of nil
func TryToUnwrap(err error) error {
	if err == nil {
		return nil
	}

	if e, ok := err.(*Error); ok {
		return TryToUnwrap(e.Err)
	}

	if chainable, ok := err.(Chainable); ok {
		chain := chainable.Chain()
		return TryToUnwrap(chain[len(chain)-1])
	}

	return err
}

// Dive into wrapped errors from src until return an error that casts to
// the target error or such error is not found
// nolint:golint
func ErrorAs(src error, target error) (error, bool) {
	if target == nil {
		return nil, false
	}

	targetValue := reflect.ValueOf(target)

	r := src

	for r != nil {
		srcValue := reflect.ValueOf(r)

		if srcValue.Type() == targetValue.Type() {
			return r, true
		}

		r = errors.Unwrap(r)
	}

	return nil, false
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

func errInChain(err error) string {
	e, ok := err.(*Error)

	if !ok {
		return err.Error()
	}

	if len(e.Msg) > 0 {
		return fmt.Sprintf("error:%v;file%s;fileline:%d", e.Msg, e.Filename, e.Line)
	}

	return fmt.Sprintf("%s:%d", e.Filename, e.Line)
}

func (chain ErrorChain) Error() string {
	s := ""

	for _, e := range chain {
		s += " > " + errInChain(e)
	}

	return s
}

type JsonErrorObject struct {
	Error string `json:"error,omitempty"`
	File  string `json:"file,omitempty"`
	Line  int    `json:"line,omitempty"`
}

func (chain ErrorChain) JSON() []JsonErrorObject {
	errs := make([]JsonErrorObject, len(chain))

	for i, e := range chain {
		errs[i] = errInJsonChain(e)
	}

	return errs
}

func errInJsonChain(err error) JsonErrorObject {
	e, ok := err.(*Error)
	if !ok {
		return JsonErrorObject{Error: err.Error()}
	}

	if len(e.Msg) > 0 {
		return JsonErrorObject{Error: e.Msg, File: e.Filename, Line: e.Line}
	}

	return JsonErrorObject{File: e.Filename, Line: e.Line}
}
